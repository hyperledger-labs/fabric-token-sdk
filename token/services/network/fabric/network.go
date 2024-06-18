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

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/services/chaincode"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/events"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	common2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/keys"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	tokens2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

const (
	InvokeFunction            = "invoke"
	QueryPublicParamsFunction = "queryPublicParams"
	QueryTokensFunctions      = "queryTokens"
	AreTokensSpent            = "areTokensSpent"
	maxRetries                = 3
	retryWaitDuration         = 1 * time.Second
)

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

type nv struct {
	v          *fabric.Vault
	tokenVault driver.TokenVault
}

func (v *nv) QueryEngine() vault.QueryEngine {
	return v.tokenVault.QueryEngine()
}

func (v *nv) CertificationStorage() vault.CertificationStorage {
	return v.tokenVault.CertificationStorage()
}

func (v *nv) DeleteTokens(ids ...*token.ID) error {
	return v.tokenVault.DeleteTokens(ids...)
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

type ledger struct {
	l *fabric.Ledger
}

func (l *ledger) Status(ctx context.Context, id string) (driver.ValidationCode, error) {
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

type Network struct {
	n              *fabric.NetworkService
	ch             *fabric.Channel
	tmsProvider    *token2.ManagementServiceProvider
	viewManager    *view2.Manager
	ledger         *ledger
	nsFinder       common2.NamespaceFinder
	filterProvider common2.TransactionFilterProvider[*common2.AcceptTxInDBsFilter]
	tokensProvider *tokens2.Manager

	vaultLazyCache utils.LazyProvider[string, driver.Vault]
	subscribers    *events.Subscribers
}

func NewNetwork(
	sp token2.ServiceProvider,
	n *fabric.NetworkService,
	ch *fabric.Channel,
	newVault NewVaultFunc,
	nsFinder common2.NamespaceFinder,
	filterProvider common2.TransactionFilterProvider[*common2.AcceptTxInDBsFilter],
	tokensProvider *tokens2.Manager,
) *Network {
	loader := &loader{
		newVault: newVault,
		name:     n.Name(),
		channel:  ch.Name(),
		vault:    ch.Vault(),
	}
	return &Network{
		n:              n,
		ch:             ch,
		nsFinder:       nsFinder,
		filterProvider: filterProvider,
		tokensProvider: tokensProvider,
		tmsProvider:    token2.GetManagementServiceProvider(sp),
		viewManager:    view2.GetManager(sp),
		ledger:         &ledger{ch.Ledger()},
		subscribers:    events.NewSubscribers(),
		vaultLazyCache: utils.NewLazyProvider(loader.load),
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
		if ns, err := n.nsFinder.LookupNamespace(opt.Network, opt.Channel); err == nil {
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
			utils.NewLazyGetter[*tokens2.Tokens](func() (*tokens2.Tokens, error) {
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
	if err := n.ch.Committer().AddTransactionFilter(transactionFilter); err != nil {
		return nil, errors.WithMessagef(err, "failed to fetch attach transaction filter [%s]", tmsID)
	}

	// check the vault for public parameters,
	// use them if they exists
	v, err := n.Vault(ns)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get network at [%s]", tmsID)
	}
	ppRaw, err := v.QueryEngine().PublicParams()
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get public params at [%s]", tmsID)
	}
	if len(ppRaw) != 0 {
		return []token2.ServiceOption{token2.WithTMSID(tmsID), token2.WithPublicParameter(ppRaw)}, nil
	}
	return []token2.ServiceOption{token2.WithTMSID(tmsID)}, nil
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
	return driver.TransientMap(tm), nil
}

func (n *Network) RequestApproval(context view.Context, tms *token2.ManagementService, requestRaw []byte, signer view.Identity, txID driver.TxID) (driver.Envelope, error) {
	env, err := chaincode.NewEndorseView(
		tms.Namespace(),
		InvokeFunction,
	).WithNetwork(
		n.n.Name(),
	).WithChannel(
		n.ch.Name(),
	).WithSignerIdentity(
		signer,
	).WithTransientEntry(
		"token_request", requestRaw,
	).WithTxID(
		fabric.TxID{
			Nonce:   txID.Nonce,
			Creator: txID.Creator,
		},
	).Endorse(context)
	if err != nil {
		return nil, err
	}
	return env, nil
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

func (n *Network) FetchPublicParameters(ctx context.Context, namespace string) ([]byte, error) {
	ppBoxed, err := n.viewManager.InitiateView(
		chaincode.NewQueryView(
			namespace,
			QueryPublicParamsFunction,
		).WithNetwork(n.Name()).WithChannel(n.Channel()), ctx,
	)
	if err != nil {
		return nil, err
	}
	return ppBoxed.([]byte), nil
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

func (n *Network) LookupTransferMetadataKey(namespace string, startingTxID string, key string, timeout time.Duration, stopOnLastTx bool, ctx context.Context) ([]byte, error) {
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

		rws, err := v.GetEphemeralRWSet(tx.Results())
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
}

func (t *FinalityListener) OnStatus(txID string, status int, message string) {
	defer func() {
		if e := recover(); e != nil {
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
	var tokenRequestHash []byte
	var retries int
	for tokenRequestHash, err = qe.GetState(t.namespace, key); err == nil && len(tokenRequestHash) == 0 && retries < maxRetries; tokenRequestHash, err = qe.GetState(t.namespace, key) {
		logger.Debugf("did not find token request [%s]. retrying...", txID)
		retries++
		time.Sleep(retryWaitDuration)
	}
	if err != nil {
		panic(fmt.Sprintf("can't get state [%s][%s]", txID, key))
	}
	if err != nil {
		qe.Done()
	}
	qe.Done()
	t.root.OnStatus(txID, status, message, tokenRequestHash)
}

type loader struct {
	newVault NewVaultFunc
	name     string
	channel  string
	vault    *fabric.Vault
}

func (l *loader) load(namespace string) (driver.Vault, error) {
	tokenVault, err := l.newVault(l.name, l.channel, namespace)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get token vault")
	}
	return &nv{v: l.vault, tokenVault: tokenVault}, nil
}
