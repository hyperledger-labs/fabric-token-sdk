/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabric

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/lazy"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/services/chaincode"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/events"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracing"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	driver3 "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	common2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/keys"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/endorsement"
	tokens2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap/zapcore"
)

const (
	QueryPublicParamsFunction = "queryPublicParams"
	QueryTokensFunctions      = "queryTokens"
	AreTokensSpent            = "areTokensSpent"
	maxRetries                = 3
	retryWaitDuration         = 1 * time.Second
)

var logger = logging.MustGetLogger("token-sdk.network.fabric")

type NewVaultFunc = func(network, channel, namespace string) (vault.Vault, error)

type lm struct {
	lm *fabric.LocalMembership
}

func (n *lm) DefaultIdentity() view.Identity {
	return n.lm.DefaultIdentity()
}

func (n *lm) AnonymousIdentity() view.Identity {
	return n.lm.AnonymousIdentity()
}

type qe struct {
	driver2.QueryExecutor
	ns string
}

func (q *qe) GetState(id token.ID) ([]byte, error) {
	key, err := keys.CreateTokenKey(id.TxId, id.Index)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting token key for [%v]", id)
	}
	return q.QueryExecutor.GetState(q.ns, key)
}

func (q *qe) Done() {
	q.QueryExecutor.Done()
}

type nv struct {
	v  *fabric.Vault
	ns string
}

func (v *nv) Status(id string) (driver.ValidationCode, string, error) {
	vc, message, err := v.v.Status(id)
	return vc, message, err
}

func (v *nv) GetLastTxID() (string, error) {
	return v.v.GetLastTxID()
}

func (v *nv) DiscardTx(id string, message string) error {
	return v.v.DiscardTx(id, message)
}

func (v *nv) NewQueryExecutor() (driver.QueryExecutor, error) {
	qeInstance, err := v.v.NewQueryExecutor()
	if err != nil {
		return nil, errors.WithMessagef(err, "failed creating query executor")
	}
	return &qe{QueryExecutor: qeInstance, ns: v.ns}, nil
}

type tokenVault struct {
	tokenVault driver.TokenVault
}

func (v *tokenVault) QueryEngine() vault.QueryEngine {
	return v.tokenVault.QueryEngine()
}

func (v *tokenVault) CertificationStorage() vault.CertificationStorage {
	return v.tokenVault.CertificationStorage()
}

func (v *tokenVault) DeleteTokens(ids ...*token.ID) error {
	return v.tokenVault.DeleteTokens(ids...)
}

type ledger struct {
	l *fabric.Ledger
}

func (l *ledger) Status(id string) (driver.ValidationCode, error) {
	tx, err := l.l.GetTransactionByID(id)
	if err != nil {
		return driver.Unknown, errors.Wrapf(err, "failed to get transaction [%s]", id)
	}
	logger.Debugf("ledger status of [%s] is [%d]", id, tx.ValidationCode())
	switch peer.TxValidationCode(tx.ValidationCode()) {
	case peer.TxValidationCode_VALID:
		return driver.Valid, nil
	default:
		return driver.Invalid, nil
	}
}

type ViewManager interface {
	InitiateView(view view2.View, ctx context.Context) (interface{}, error)
}

type ViewRegistry interface {
	RegisterResponder(responder view.View, initiatedBy interface{}) error
}

type Network struct {
	n              *fabric.NetworkService
	ch             *fabric.Channel
	tmsProvider    *token2.ManagementServiceProvider
	viewManager    ViewManager
	ledger         *ledger
	configuration  common2.Configuration
	filterProvider common2.TransactionFilterProvider[*common2.AcceptTxInDBsFilter]
	tokensProvider *tokens2.Manager
	finalityTracer trace.Tracer

	vaultLazyCache             lazy.Provider[string, driver.Vault]
	tokenVaultLazyCache        lazy.Provider[string, driver.TokenVault]
	subscribers                *events.Subscribers
	defaultPublicParamsFetcher driver3.NetworkPublicParamsFetcher

	endorsementServiceProvider *endorsement.ServiceProvider
}

func NewNetwork(
	n *fabric.NetworkService,
	ch *fabric.Channel,
	newVault NewVaultFunc,
	configuration common2.Configuration,
	filterProvider common2.TransactionFilterProvider[*common2.AcceptTxInDBsFilter],
	tokensProvider *tokens2.Manager,
	viewManager ViewManager,
	tmsProvider *token2.ManagementServiceProvider,
	endorsementServiceProvider *endorsement.ServiceProvider,
	tracerProvider trace.TracerProvider,
	defaultPublicParamsFetcher driver3.NetworkPublicParamsFetcher,
) *Network {
	loader := &loader{
		newVault: newVault,
		name:     n.Name(),
		channel:  ch.Name(),
		vault:    ch.Vault(),
	}
	return &Network{
		n:                          n,
		ch:                         ch,
		tmsProvider:                tmsProvider,
		viewManager:                viewManager,
		ledger:                     &ledger{l: ch.Ledger()},
		configuration:              configuration,
		filterProvider:             filterProvider,
		tokensProvider:             tokensProvider,
		vaultLazyCache:             lazy.NewProvider(loader.loadVault),
		tokenVaultLazyCache:        lazy.NewProvider(loader.loadTokenVault),
		subscribers:                events.NewSubscribers(),
		defaultPublicParamsFetcher: defaultPublicParamsFetcher,
		endorsementServiceProvider: endorsementServiceProvider,
		finalityTracer: tracerProvider.Tracer("finality_listener", tracing.WithMetricsOpts(tracing.MetricsOpts{
			Namespace:  "tokensdk_fabric",
			LabelNames: []tracing.LabelName{},
		})),
	}
}

func (n *Network) Name() string {
	return n.n.Name()
}

func (n *Network) Channel() string {
	return n.ch.Name()
}

func (n *Network) Normalize(opt *token2.ServiceOptions) (*token2.ServiceOptions, error) {
	if len(opt.Network) == 0 {
		opt.Network = n.n.Name()
	}
	if opt.Network != n.n.Name() {
		return nil, errors.Errorf("invalid network [%s], expected [%s]", opt.Network, n.n.Name())
	}

	if len(opt.Channel) == 0 {
		opt.Channel = n.ch.Name()
	}
	if opt.Channel != n.ch.Name() {
		return nil, errors.Errorf("invalid channel [%s], expected [%s]", opt.Channel, n.ch.Name())
	}

	if len(opt.Namespace) == 0 {
		if ns, err := n.configuration.LookupNamespace(opt.Network, opt.Channel); err == nil {
			logger.Debugf("no namespace specified, found namespace [%s] for [%s:%s]", ns, opt.Network, opt.Channel)
			opt.Namespace = ns
		} else {
			logger.Errorf("no namespace specified, and no default namespace found [%s], use default [%s]", err, ttx.TokenNamespace)
			opt.Namespace = ttx.TokenNamespace
		}
	}
	if opt.PublicParamsFetcher == nil {
		opt.PublicParamsFetcher = common2.NewPublicParamsFetcher(n, opt.Namespace)
	}
	return opt, nil
}

func (n *Network) Connect(ns string) ([]token2.ServiceOption, error) {
	tmsID := token2.TMSID{
		Network:   n.n.Name(),
		Channel:   n.ch.Name(),
		Namespace: ns,
	}

	if err := n.n.ProcessorManager().AddProcessor(
		ns,
		NewTokenRWSetProcessor(
			n.Name(),
			ns,
			lazy.NewGetter[*tokens2.Tokens](func() (*tokens2.Tokens, error) {
				return n.tokensProvider.Tokens(tmsID)
			}).Get,
			func() *token2.ManagementServiceProvider {
				return n.tmsProvider
			},
		)); err != nil {
		return nil, errors.WithMessagef(err, "failed to add processor to fabric network [%s]", n.n.Name())
	}
	transactionFilter, err := n.filterProvider.New(tmsID)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to create transaction filter for [%s]", tmsID)
	}
	committer := n.ch.Committer()
	if err := committer.AddTransactionFilter(transactionFilter); err != nil {
		return nil, errors.WithMessagef(err, "failed to fetch attach transaction filter [%s]", tmsID)
	}

	// Let the endorsement service initialize itself, if needed
	_, err = n.endorsementServiceProvider.Get(tmsID)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get endorsement service at [%s]", tmsID)
	}
	return nil, nil
}

func (n *Network) Vault(namespace string) (driver.Vault, error) {
	if len(namespace) == 0 {
		tms, err := n.tmsProvider.GetManagementService(token2.WithNetwork(n.n.Name()), token2.WithChannel(n.ch.Name()))
		if tms == nil || err != nil {
			return nil, errors.Errorf("empty namespace passed, cannot find TMS for [%s:%s]: %v", n.n.Name(), n.ch.Name(), err)
		}
		namespace = tms.Namespace()
	}
	return n.vaultLazyCache.Get(namespace)
}

func (n *Network) TokenVault(namespace string) (driver.TokenVault, error) {
	if len(namespace) == 0 {
		tms, err := n.tmsProvider.GetManagementService(token2.WithNetwork(n.n.Name()), token2.WithChannel(n.ch.Name()))
		if tms == nil || err != nil {
			return nil, errors.Errorf("empty namespace passed, cannot find TMS for [%s:%s]: %v", n.n.Name(), n.ch.Name(), err)
		}
		namespace = tms.Namespace()
	}
	return n.tokenVaultLazyCache.Get(namespace)
}

func (n *Network) Broadcast(context context.Context, blob interface{}) error {
	return n.n.Ordering().Broadcast(context, blob)
}

func (n *Network) NewEnvelope() driver.Envelope {
	return n.n.TransactionManager().NewEnvelope()
}

func (n *Network) StoreTransient(id string, transient driver.TransientMap) error {
	return n.ch.Vault().StoreTransient(id, transient)
}

func (n *Network) TransientExists(id string) bool {
	return n.ch.MetadataService().Exists(id)
}

func (n *Network) GetTransient(id string) (driver.TransientMap, error) {
	tm, err := n.ch.MetadataService().LoadTransient(id)
	if err != nil {
		return nil, err
	}
	return tm, nil
}

func (n *Network) RequestApproval(context view.Context, tms *token2.ManagementService, requestRaw []byte, signer view.Identity, txID driver.TxID) (driver.Envelope, error) {
	endorsement, err := n.endorsementServiceProvider.Get(tms.ID())
	if err != nil {
		return nil, errors.Wrapf(err, "network not connected [%s]", tms.ID())
	}
	return endorsement.Endorse(context, requestRaw, signer, txID)
}

func (n *Network) ComputeTxID(id *driver.TxID) string {
	logger.Debugf("compute tx id for [%s]", id.String())
	temp := &fabric.TxID{
		Nonce:   id.Nonce,
		Creator: id.Creator,
	}
	res := n.n.TransactionManager().ComputeTxID(temp)
	id.Nonce = temp.Nonce
	id.Creator = temp.Creator
	return res
}

func (n *Network) FetchPublicParameters(namespace string) ([]byte, error) {
	return n.defaultPublicParamsFetcher.Fetch(n.Name(), n.Channel(), namespace)
}

func (n *Network) QueryTokens(context view.Context, namespace string, IDs []*token.ID) ([][]byte, error) {
	idsRaw, err := json.Marshal(IDs)
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshalling ids")
	}

	payloadBoxed, err := context.RunView(chaincode.NewQueryView(
		namespace,
		QueryTokensFunctions,
		idsRaw,
	).WithNetwork(n.Name()).WithChannel(n.Channel()))
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to query the token chaincode for tokens")
	}

	// Unbox
	raw, ok := payloadBoxed.([]byte)
	if !ok {
		return nil, errors.Errorf("expected []byte from TCC, got [%T]", payloadBoxed)
	}
	var tokens [][]byte
	if err := json.Unmarshal(raw, &tokens); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal response")
	}

	return tokens, nil
}

func (n *Network) AreTokensSpent(c view.Context, namespace string, tokenIDs []*token.ID, meta []string) ([]bool, error) {
	sIDs := make([]string, len(tokenIDs))
	var err error
	for i, id := range tokenIDs {
		sIDs[i], err = keys.CreateTokenKey(id.TxId, id.Index)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot compute spent id for [%v]", id)
		}
	}

	idsRaw, err := json.Marshal(sIDs)
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshalling ids")
	}

	payloadBoxed, err := c.RunView(chaincode.NewQueryView(
		namespace,
		AreTokensSpent,
		idsRaw,
	).WithNetwork(n.Name()).WithChannel(n.Channel()))
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to query the token chaincode for tokens spent")
	}

	// Unbox
	raw, ok := payloadBoxed.([]byte)
	if !ok {
		return nil, errors.Errorf("expected []byte from TCC, got [%T]", payloadBoxed)
	}
	var spent []bool
	if err := json.Unmarshal(raw, &spent); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal esponse")
	}

	return spent, nil
}

func (n *Network) LocalMembership() driver.LocalMembership {
	return &lm{
		lm: n.n.LocalMembership(),
	}
}

func (n *Network) AddFinalityListener(namespace string, txID string, listener driver.FinalityListener) error {
	wrapper := &FinalityListener{
		net:       n,
		root:      listener,
		network:   n.n.Name(),
		namespace: namespace,
		tracer:    n.finalityTracer,
	}
	n.subscribers.Set(txID, listener, wrapper)
	return n.ch.Committer().AddFinalityListener(txID, wrapper)
}

func (n *Network) RemoveFinalityListener(txID string, listener driver.FinalityListener) error {
	wrapper, ok := n.subscribers.Get(txID, listener)
	if !ok {
		return errors.Errorf("listener was not registered")
	}
	return n.ch.Committer().RemoveFinalityListener(txID, wrapper.(*FinalityListener))
}

func (n *Network) LookupTransferMetadataKey(namespace string, startingTxID string, key string, timeout time.Duration, stopOnLastTx bool) ([]byte, error) {
	transferMetadataKey, err := keys.CreateTransferActionMetadataKey(key)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to generate transfer action metadata key from [%s]", key)
	}
	var keyValue []byte
	c, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	v := n.ch.Vault()

	var lastTxID string
	if stopOnLastTx {
		id, err := v.GetLastTxID()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get last transaction id")
		}
		lastTxID = id
	}

	if err := n.ch.Delivery().Scan(c, startingTxID, func(tx *fabric.ProcessedTransaction) (bool, error) {
		logger.Debugf("scanning [%s]...", tx.TxID())

		rws, err := v.InspectRWSet(tx.Results())
		if err != nil {
			return false, err
		}

		found := false
		for _, ns := range rws.Namespaces() {
			if ns == namespace {
				found = true
				break
			}
		}
		if !found {
			logger.Debugf("scanning [%s] does not contain namespace [%s]", tx.TxID(), namespace)
			return false, nil
		}

		ns := namespace
		for i := 0; i < rws.NumWrites(ns); i++ {
			k, v, err := rws.GetWriteAt(ns, i)
			if err != nil {
				return false, err
			}
			if k == transferMetadataKey {
				keyValue = v
				return true, nil
			}
		}
		logger.Debugf("scanning for key [%s] on [%s] not found", transferMetadataKey, tx.TxID())
		if stopOnLastTx && lastTxID == tx.TxID() {
			logger.Debugf("final transaction reached on [%s]", tx.TxID())
			cancel()
		}

		return false, nil
	}); err != nil {
		if strings.Contains(err.Error(), "context done") {
			return nil, errors.WithMessage(err, "timeout reached")
		}
		return nil, err
	}

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("scanning for key [%s] with timeout [%s] found, [%s]",
			transferMetadataKey,
			timeout,
			base64.StdEncoding.EncodeToString(keyValue),
		)
	}
	return keyValue, nil
}

func (n *Network) Ledger() (driver.Ledger, error) {
	return n.ledger, nil
}

func (n *Network) ProcessNamespace(namespace string) error {
	if err := n.ch.Committer().ProcessNamespace(namespace); err != nil {
		return errors.WithMessagef(err, "failed to register processing of namespace [%s]", namespace)
	}
	return nil
}

type FinalityListener struct {
	net       *Network
	root      driver.FinalityListener
	network   string
	namespace string
	tracer    trace.Tracer
}

func (t *FinalityListener) OnStatus(ctx context.Context, txID string, status int, message string) {
	newCtx, span := t.tracer.Start(ctx, "on_status")
	defer span.End()
	defer func() {
		if e := recover(); e != nil {
			span.RecordError(fmt.Errorf("recovered from panic: %v", e))
			logger.Debugf("failed finality update for tx [%s]: [%s]", txID, e)
			if err := t.net.AddFinalityListener(txID, t.namespace, t.root); err != nil {
				panic(err)
			}
			logger.Debugf("added finality listener for tx [%s]...done", txID)
		}
	}()

	key, err := keys.CreateTokenRequestKey(txID)
	if err != nil {
		panic(fmt.Sprintf("can't create for token request [%s]", txID))
	}

	v := t.net.ch.Vault()
	qe, err := v.NewQueryExecutor()
	if err != nil {
		panic(fmt.Sprintf("can't get query executor [%s]", txID))
	}

	// Fetch the token request hash. Retry in case some other replica committed it shortly before
	span.AddEvent("fetch_request_hash")
	var tokenRequestHash []byte
	var retries int
	for tokenRequestHash, err = qe.GetState(t.namespace, key); err == nil && len(tokenRequestHash) == 0 && retries < maxRetries; tokenRequestHash, err = qe.GetState(t.namespace, key) {
		span.AddEvent("retry_fetch_request_hash")
		logger.Debugf("did not find token request [%s]. retrying...", txID)
		retries++
		time.Sleep(retryWaitDuration)
	}
	qe.Done()
	if err != nil {
		panic(fmt.Sprintf("can't get state [%s][%s]", txID, key))
	}
	span.AddEvent("call_root_listener")
	t.root.OnStatus(newCtx, txID, status, message, tokenRequestHash)
}

type loader struct {
	newVault NewVaultFunc
	name     string
	channel  string
	vault    *fabric.Vault
}

func (l *loader) loadVault(namespace string) (driver.Vault, error) {
	return &nv{v: l.vault, ns: namespace}, nil
}

func (l *loader) loadTokenVault(namespace string) (driver.TokenVault, error) {
	tv, err := l.newVault(l.name, l.channel, namespace)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get token vault")
	}
	return &tokenVault{tokenVault: tv}, nil
}
