/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabric

import (
	"context"
	"sync"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/lazy"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracing"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	common2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/translator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/endorsement"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/finality"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/lookup"
	tokens2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
)

const (
	QueryPublicParamsFunction = "queryPublicParams"
	QueryTokensFunctions      = "queryTokens"
	AreTokensSpent            = "areTokensSpent"
)

var logger = logging.MustGetLogger("token-sdk.network.fabric")

type NewVaultFunc = func(network, channel, namespace string) (driver.TokenVault, error)

type lm struct {
	lm *fabric.LocalMembership
}

func (n *lm) DefaultIdentity() view.Identity {
	return n.lm.DefaultIdentity()
}

func (n *lm) AnonymousIdentity() (view.Identity, error) {
	return n.lm.AnonymousIdentity()
}

type tokenVault struct {
	tokenVault driver.TokenVault
}

func (v *tokenVault) QueryEngine() driver.QueryEngine {
	return v.tokenVault.QueryEngine()
}

func (v *tokenVault) CertificationStorage() driver.CertificationStorage {
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

type EndorsementService = endorsement.Service

type EndorsementServiceProvider = lazy.Provider[token2.TMSID, EndorsementService]

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

	tokenVaultLazyCache        lazy.Provider[string, driver.TokenVault]
	flm                        finality.ListenerManager
	llm                        lookup.ListenerManager
	defaultPublicParamsFetcher NetworkPublicParamsFetcher
	tokenQueryExecutor         driver.TokenQueryExecutor
	spentTokenQueryExecutor    driver.SpentTokenQueryExecutor
	endorsementServiceProvider EndorsementServiceProvider
	keyTranslator              translator.KeyTranslator
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
	endorsementServiceProvider EndorsementServiceProvider,
	tokenQueryExecutor driver.TokenQueryExecutor,
	tracerProvider trace.TracerProvider,
	defaultPublicParamsFetcher NetworkPublicParamsFetcher,
	spentTokenQueryExecutor driver.SpentTokenQueryExecutor,
	keyTranslator translator.KeyTranslator,
	flm finality.ListenerManager,
	llm lookup.ListenerManager,
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
		tokenVaultLazyCache:        lazy.NewProvider(loader.loadTokenVault),
		flm:                        flm,
		llm:                        llm,
		defaultPublicParamsFetcher: defaultPublicParamsFetcher,
		endorsementServiceProvider: endorsementServiceProvider,
		tokenQueryExecutor:         tokenQueryExecutor,
		spentTokenQueryExecutor:    spentTokenQueryExecutor,
		finalityTracer: tracerProvider.Tracer("finality_listener", tracing.WithMetricsOpts(tracing.MetricsOpts{
			Namespace:  "tokensdk_fabric",
			LabelNames: []tracing.LabelName{},
		})),
		keyTranslator: keyTranslator,
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

	if n.llm.PermanentLookupListenerSupported() {
		setUpKey, err := n.keyTranslator.CreateSetupKey()
		if err != nil {
			return nil, errors.Errorf("failed creating setup key")
		}
		if err := n.llm.AddPermanentLookupListener(ns, setUpKey, &setupListener{
			GetTMSProvider: func() *token2.ManagementServiceProvider {
				return n.tmsProvider
			},
			GetTokens: lazy.NewGetter[*tokens2.Tokens](func() (*tokens2.Tokens, error) {
				return n.tokensProvider.Tokens(tmsID)
			}).Get,
			TMSID: token2.TMSID{
				Network:   n.Name(),
				Channel:   n.Channel(),
				Namespace: ns,
			},
		}); err != nil {
			return nil, errors.Errorf("failed adding setup key listener")
		}
	} else {
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
				n.keyTranslator,
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
	}

	// Let the endorsement service initialize itself, if needed
	if _, err := n.endorsementServiceProvider.Get(tmsID); err != nil {
		return nil, errors.WithMessagef(err, "failed to get endorsement service at [%s]", tmsID)
	}
	return nil, nil
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

func (n *Network) Broadcast(ctx context.Context, blob interface{}) error {
	return n.n.Ordering().Broadcast(ctx, blob)
}

func (n *Network) NewEnvelope() driver.Envelope {
	return n.n.TransactionManager().NewEnvelope()
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

func (n *Network) QueryTokens(ctx context.Context, namespace string, IDs []*token.ID) ([][]byte, error) {
	return n.tokenQueryExecutor.QueryTokens(ctx, namespace, IDs)
}

func (n *Network) AreTokensSpent(ctx context.Context, namespace string, tokenIDs []*token.ID, meta []string) ([]bool, error) {
	return n.spentTokenQueryExecutor.QuerySpentTokens(ctx, namespace, tokenIDs, meta)
}

func (n *Network) LocalMembership() driver.LocalMembership {
	return &lm{
		lm: n.n.LocalMembership(),
	}
}

func (n *Network) AddFinalityListener(namespace string, txID string, listener driver.FinalityListener) error {
	return n.flm.AddFinalityListener(namespace, txID, listener)
}

func (n *Network) RemoveFinalityListener(txID string, listener driver.FinalityListener) error {
	return n.flm.RemoveFinalityListener(txID, listener)
}

func (n *Network) LookupTransferMetadataKey(namespace string, startingTxID string, key string, timeout time.Duration, stopOnLastTx bool) ([]byte, error) {
	transferMetadataKey, err := n.keyTranslator.CreateTransferActionMetadataKey(key)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to generate transfer action metadata key from [%s]", key)
	}
	logger.Debugf("lookup transfer metadata key [%s] from [%s] in namespace [%s]", key, transferMetadataKey, namespace)
	wg := &sync.WaitGroup{}
	wg.Add(1)
	l := &lookupListener{wg: wg, key: transferMetadataKey}
	if err := n.llm.AddLookupListener(namespace, transferMetadataKey, startingTxID, stopOnLastTx, l); err != nil {
		return nil, errors.Wrapf(err, "failed to add lookup listener")
	}
	defer func() {
		if err := n.llm.RemoveLookupListener(transferMetadataKey, l); err != nil {
			logger.Debugf("failed to remove lookup listener [%s]: %v", transferMetadataKey, err)
		}
	}()
	if err := waitTimeout(wg, timeout); err != nil {
		return nil, err
	}
	logger.Debugf("lookup transfer metadata key [%s] from [%s] in namespace [%s], done, result [%s][%s]", key, transferMetadataKey, namespace, l.value, l.err)
	return l.value, l.err
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

type loader struct {
	newVault NewVaultFunc
	name     string
	channel  string
	vault    *fabric.Vault
}

func (l *loader) loadTokenVault(namespace string) (driver.TokenVault, error) {
	tv, err := l.newVault(l.name, l.channel, namespace)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get token vault")
	}
	return &tokenVault{tokenVault: tv}, nil
}

type lookupListener struct {
	key   string
	wg    *sync.WaitGroup
	value []byte
	err   error
}

func (l *lookupListener) OnStatus(ctx context.Context, key string, value []byte) {
	logger.Debugf("lookup transfer metadata key [%s], got value [%s][%v]", l.key, key, value)
	if l.key == key {
		l.value = value
		l.wg.Done()
		return
	}
}

func (l *lookupListener) OnError(ctx context.Context, key string, err error) {
	logger.Debugf("lookup transfer metadata key [%s], got error [%s][%s]", l.key, key, err)
	if l.key == key {
		l.err = err
		l.wg.Done()
		return
	}
}

func waitTimeout(wg *sync.WaitGroup, timeout time.Duration) error {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		return nil
	case <-time.After(timeout):
		return errors.Errorf("context done")
	}
}

type setupListener struct {
	GetTMSProvider GetTMSProviderFunc
	GetTokens      GetTokensFunc
	TMSID          token2.TMSID
}

func (s *setupListener) OnStatus(ctx context.Context, key string, value []byte) {
	logger.Infof("update TMS [%s] with key-value [%s][%s]", s.TMSID, key, utils.Hashable(value))
	tsmProvider := s.GetTMSProvider()
	if err := tsmProvider.Update(s.TMSID, value); err != nil {
		logger.Warnf("failed to update TMS [%s]: [%v]", key, err)
	}
	tokens, err := s.GetTokens()
	if err != nil {
		logger.Warnf("failed to get tokens db [%v]", err)
	}
	if err := tokens.StorePublicParams(value); err != nil {
		logger.Warnf("failed to store public parameter key [%s]: [%v]", key, err)
	}
}

func (s *setupListener) OnError(ctx context.Context, key string, err error) {
	// TODO implement me
	panic("implement me")
}
