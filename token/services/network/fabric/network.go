/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabric

import (
	"context"
	"sync"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/lazy"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracing"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/encoding/json"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	common2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/recovery"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/translator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/endorsement"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/finality"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/lookup"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/ttxdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx/dep"
	ttxfinality "github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx/finality"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/hyperledger/fabric-protos-go-apiv2/peer"
	"go.opentelemetry.io/otel/trace"
)

const (
	// QueryPublicParamsFunction is the chaincode function name for querying public parameters.
	QueryPublicParamsFunction = "queryPublicParams"
	// QueryTokensFunctions is the chaincode function name for querying token data.
	QueryTokensFunctions = "queryTokens"
	// AreTokensSpent is the chaincode function name for checking if tokens are spent.
	AreTokensSpent = "areTokensSpent"
)

var logger = logging.MustGetLogger()

// GetTokensFunc is a function type that returns a token Service instance.
type GetTokensFunc = func() (*tokens.Service, error)

// GetTMSProviderFunc is a function type that returns a ManagementServiceProvider.
type GetTMSProviderFunc = func() *token2.ManagementServiceProvider

type lm struct {
	lm *fabric.LocalMembership
}

func (n *lm) DefaultIdentity() view.Identity {
	return n.lm.DefaultIdentity()
}

func (n *lm) AnonymousIdentity() (view.Identity, error) {
	return n.lm.AnonymousIdentity()
}

// ledger provides access to the Fabric ledger via the FSC fabric layer.
type ledger struct {
	l             *fabric.Ledger
	ch            *fabric.Channel
	keyTranslator translator.KeyTranslator
}

func newLedger(ch *fabric.Channel, keyTranslator translator.KeyTranslator) *ledger {
	return &ledger{ch: ch, l: ch.Ledger(), keyTranslator: keyTranslator}
}

// Status retrieves the validation status of a transaction from the Fabric ledger.
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

// GetStates performs a multi-key state query against the token chaincode.
func (l *ledger) GetStates(ctx context.Context, namespace string, keys ...string) ([][]byte, error) {
	if len(keys) == 0 {
		return nil, errors.Errorf("keys cannot be empty")
	}
	arg, err := json.Marshal(keys)
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshalling args for query by ids [%v]", keys)
	}
	logger.DebugfContext(ctx, "querying chaincode [%s] for the states of ids [%v]", namespace, keys)
	chaincode := l.ch.Chaincode(namespace)
	res, err := chaincode.Query(lookup.QueryStates, arg).Query()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query for states of ids [%v]", keys)
	}
	var values [][]byte
	err = json.Unmarshal(res, &values)
	if err != nil {
		return nil, errors.Wrapf(err, "failed unmarshalling results for query by ids [%v", keys)
	}

	return values, nil
}

func (l *ledger) TransferMetadataKey(k string) (string, error) {
	return l.keyTranslator.CreateTransferActionMetadataKey(k)
}

// ViewManager models the interface for initiating FSC views.
type ViewManager interface {
	InitiateView(ctx context.Context, view view.View) (interface{}, error)
}

// ViewRegistry models the interface for registering view responders.
type ViewRegistry interface {
	RegisterResponder(responder view.View, initiatedBy interface{}) error
}

// EndorsementService models the interface for transaction endorsement.
type EndorsementService = endorsement.Service

// EndorsementServiceProvider provides endorsement services for different TMS IDs.
type EndorsementServiceProvider = lazy.Provider[token2.TMSID, EndorsementService]

// SetupListenerProvider defines the interface for obtaining lookup listeners for public parameters setup.
type SetupListenerProvider interface {
	GetListener(token2.TMSID) lookup.Listener
}

// Network implements the driver.Network interface for Hyperledger Fabric.
// It orchestrates finality listeners, state queries, and endorsement requests.
type Network struct {
	n                   *fabric.NetworkService
	ch                  *fabric.Channel
	tmsProvider         *token2.ManagementServiceProvider
	viewManager         ViewManager
	ledger              *ledger
	configuration       common2.Configuration
	filterProvider      common2.TransactionFilterProvider[*common2.AcceptTxInDBsFilter]
	tokensProvider      *tokens.ServiceManager
	finalityTracer      trace.Tracer
	localMembership     *lm
	storeServiceManager ttxdb.StoreServiceManager

	setupListenerProvider      SetupListenerProvider
	flm                        finality.ListenerManager
	llm                        lookup.ListenerManager
	defaultPublicParamsFetcher NetworkPublicParamsFetcher
	tokenQueryExecutor         driver.TokenQueryExecutor
	spentTokenQueryExecutor    driver.SpentTokenQueryExecutor
	endorsementServiceProvider EndorsementServiceProvider
	keyTranslator              translator.KeyTranslator

	connectedNamespaces lazy.Provider[string, []token2.ServiceOption]
	recoveryManagers    lazy.Provider[string, *recovery.Manager]
}

// NewNetwork creates a new Fabric Network instance.
func NewNetwork(
	n *fabric.NetworkService,
	ch *fabric.Channel,
	configuration common2.Configuration,
	filterProvider common2.TransactionFilterProvider[*common2.AcceptTxInDBsFilter],
	tokensProvider *tokens.ServiceManager,
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
	setupListenerProvider SetupListenerProvider,
	storeServiceManager ttxdb.StoreServiceManager,
) *Network {
	network := &Network{
		n:                          n,
		ch:                         ch,
		tmsProvider:                tmsProvider,
		viewManager:                viewManager,
		ledger:                     newLedger(ch, keyTranslator),
		configuration:              configuration,
		filterProvider:             filterProvider,
		tokensProvider:             tokensProvider,
		flm:                        flm,
		llm:                        llm,
		defaultPublicParamsFetcher: defaultPublicParamsFetcher,
		endorsementServiceProvider: endorsementServiceProvider,
		tokenQueryExecutor:         tokenQueryExecutor,
		spentTokenQueryExecutor:    spentTokenQueryExecutor,
		finalityTracer: tracerProvider.Tracer("finality_listener", tracing.WithMetricsOpts(tracing.MetricsOpts{
			LabelNames: []tracing.LabelName{},
		})),
		keyTranslator:         keyTranslator,
		setupListenerProvider: setupListenerProvider,
		localMembership:       &lm{lm: n.LocalMembership()},
		storeServiceManager:   storeServiceManager,
	}
	network.connectedNamespaces = lazy.NewProviderWithKeyMapper(func(s string) string {
		return s
	}, network.connect)
	network.recoveryManagers = lazy.NewProviderWithKeyMapper(func(s string) string {
		return s
	}, network.createRecoveryManager)

	return network
}

// Name returns the name of the Fabric network.
func (n *Network) Name() string {
	return n.n.Name()
}

// Channel returns the name of the Fabric channel.
func (n *Network) Channel() string {
	return n.ch.Name()
}

// Normalize ensures that network, channel, and namespace are correctly set in the options.
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

// Connect initializes listeners for public parameters and initializes the endorsement service for the namespace.
func (n *Network) Connect(ns string) (opts []token2.ServiceOption, err error) {
	return n.connectedNamespaces.Get(ns)
}

// Broadcast sends a transaction envelope to the ordering service.
func (n *Network) Broadcast(ctx context.Context, blob interface{}) error {
	return n.n.Ordering().Broadcast(ctx, blob)
}

// NewEnvelope returns a new transaction envelope for the Fabric network.
func (n *Network) NewEnvelope() driver.Envelope {
	return n.n.TransactionManager().NewEnvelope()
}

// RequestApproval requests an endorsement for a token request.
func (n *Network) RequestApproval(context view.Context, tms *token2.ManagementService, requestRaw []byte, signer view.Identity, txID driver.TxID) (driver.Envelope, error) {
	endorsement, err := n.endorsementServiceProvider.Get(tms.ID())
	if err != nil {
		return nil, errors.Wrapf(err, "network not connected [%s]", tms.ID())
	}

	return endorsement.Endorse(context, requestRaw, signer, txID)
}

// ComputeTxID calculates the Fabric transaction ID based on creator and nonce.
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

// FetchPublicParameters queries the chaincode for the current public parameters.
func (n *Network) FetchPublicParameters(namespace string) ([]byte, error) {
	return n.defaultPublicParamsFetcher.Fetch(n.Name(), n.Channel(), namespace)
}

// QueryTokens retrieves token data from the global state.
func (n *Network) QueryTokens(ctx context.Context, namespace string, IDs []*token.ID) ([][]byte, error) {
	return n.tokenQueryExecutor.QueryTokens(ctx, namespace, IDs)
}

// AreTokensSpent checks if tokens have been consumed by verifying their existence in the ledger.
func (n *Network) AreTokensSpent(ctx context.Context, namespace string, tokenIDs []*token.ID, meta []string) ([]bool, error) {
	return n.spentTokenQueryExecutor.QuerySpentTokens(ctx, namespace, tokenIDs, meta)
}

// LocalMembership returns the membership service for the local FSC node.
func (n *Network) LocalMembership() driver.LocalMembership {
	return n.localMembership
}

// AddFinalityListener registers a callback for transaction status updates.
func (n *Network) AddFinalityListener(namespace string, txID string, listener driver.FinalityListener) error {
	return n.flm.AddFinalityListener(namespace, txID, listener)
}

// LookupTransferMetadataKey performs a scan to find transfer metadata matching a sub-key.
func (n *Network) LookupTransferMetadataKey(namespace string, key string, timeout time.Duration) ([]byte, error) {
	transferMetadataKey, err := n.keyTranslator.CreateTransferActionMetadataKey(key)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to generate transfer action metadata key from [%s]", key)
	}
	logger.Debugf("lookup transfer metadata key [%s] from [%s] in namespace [%s]", key, transferMetadataKey, namespace)
	wg := &sync.WaitGroup{}
	wg.Add(1)
	l := &lookupListener{wg: wg, key: transferMetadataKey}
	if err := n.llm.AddLookupListener(namespace, transferMetadataKey, l); err != nil {
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

// Ledger returns direct access to the ledger querying layer.
func (n *Network) Ledger() (driver.Ledger, error) {
	return n.ledger, nil
}

func (n *Network) connect(ns string) ([]token2.ServiceOption, error) {
	tmsID := token2.TMSID{
		Network:   n.n.Name(),
		Channel:   n.ch.Name(),
		Namespace: ns,
	}

	setUpKey, err := n.keyTranslator.CreateSetupKey()
	if err != nil {
		return nil, errors.Errorf("failed creating setup key")
	}
	if err := n.llm.AddPermanentLookupListener(ns, setUpKey, n.setupListenerProvider.GetListener(tmsID)); err != nil {
		return nil, errors.Errorf("failed adding setup key listener")
	}

	// Let the endorsement service initialize itself, if needed
	if _, err := n.endorsementServiceProvider.Get(tmsID); err != nil {
		return nil, errors.WithMessagef(err, "failed to get endorsement service at [%s]", tmsID)
	}

	// Initialize and start recovery manager using lazy provider
	if _, err := n.recoveryManagers.Get(ns); err != nil {
		return nil, errors.WithMessagef(err, "failed to start recovery manager for [%s]", tmsID)
	}

	return nil, nil
}

// finalityListenerFactory creates finality listeners for transaction recovery
type finalityListenerFactory struct {
	networkAdapter *networkAdapter
	tmsProvider    *token2.ManagementServiceProvider
	tmsID          token2.TMSID
	ttxDB          *ttxdb.StoreService
	tokensService  *tokens.Service
	finalityTracer trace.Tracer
}

// NewFinalityListener creates a new finality listener for the given transaction ID
func (f *finalityListenerFactory) NewFinalityListener(txID string) (network.FinalityListener, error) {
	// Create a TMS provider adapter
	tmsProviderAdapter := &tmsProviderAdapter{provider: f.tmsProvider}

	// Create and return a new finality listener
	return ttxfinality.NewListener(
		logger,
		f.networkAdapter,
		f.tmsID.Namespace,
		tmsProviderAdapter,
		f.tmsID,
		f.ttxDB,
		f.tokensService,
		f.finalityTracer,
	), nil
}

// tmsProviderAdapter adapts token2.ManagementServiceProvider to dep.TokenManagementServiceProvider
type tmsProviderAdapter struct {
	provider *token2.ManagementServiceProvider
}

func (a *tmsProviderAdapter) TokenManagementService(opts ...token2.ServiceOption) (dep.TokenManagementServiceWithExtensions, error) {
	tms, err := a.provider.GetManagementService(opts...)
	if err != nil {
		return nil, err
	}

	return &tmsAdapter{tms: tms}, nil
}

// tmsAdapter adapts token2.ManagementService to ttx dep.TokenManagementServiceWithExtensions
type tmsAdapter struct {
	tms *token2.ManagementService
}

func (a *tmsAdapter) ID() token2.TMSID { return a.tms.ID() }
func (a *tmsAdapter) Network() string  { return a.tms.Network() }
func (a *tmsAdapter) Channel() string  { return a.tms.Channel() }
func (a *tmsAdapter) NewRequest(anchor token2.RequestAnchor) (*token2.Request, error) {
	return a.tms.NewRequest(anchor)
}

func (a *tmsAdapter) SelectorManager() (token2.SelectorManager, error) {
	return a.tms.SelectorManager()
}

func (a *tmsAdapter) PublicParametersManager() *token2.PublicParametersManager {
	return a.tms.PublicParametersManager()
}
func (a *tmsAdapter) SigService() *token2.SignatureService { return a.tms.SigService() }
func (a *tmsAdapter) WalletManager() *token2.WalletManager { return a.tms.WalletManager() }
func (a *tmsAdapter) NewFullRequestFromBytes(raw []byte) (*token2.Request, error) {
	return a.tms.NewFullRequestFromBytes(raw)
}
func (a *tmsAdapter) Vault() *token2.Vault { return a.tms.Vault() }
func (a *tmsAdapter) SetTokenManagementService(req *token2.Request) error {
	// This is typically a no-op for recovery scenarios
	return nil
}

// createRecoveryManager initializes and starts the recovery manager for the given namespace
func (n *Network) createRecoveryManager(ns string) (*recovery.Manager, error) {
	tmsID := token2.TMSID{
		Network:   n.n.Name(),
		Channel:   n.ch.Name(),
		Namespace: ns,
	}

	// Get TMS configuration
	cfg, err := n.configuration.ConfigurationFor(tmsID.Network, tmsID.Channel, tmsID.Namespace)
	if err != nil {
		logger.Warnf("failed to get configuration for [%s], using default recovery config: %v", tmsID, err)
		cfg = nil
	}

	// Load recovery configuration
	var recoveryConfig recovery.Config
	if cfg != nil {
		recoveryConfig, err = recovery.LoadConfig(cfg)
		if err != nil {
			logger.Warnf("failed to load recovery config for [%s], using defaults: %v", tmsID, err)
			recoveryConfig = recovery.DefaultConfig()
		}
	} else {
		recoveryConfig = recovery.DefaultConfig()
	}

	// Get tokens service
	tokensService, err := n.tokensProvider.ServiceByTMSId(tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get tokens service for [%s]", tmsID)
	}

	// Get TTXDB store service using the store service manager
	ttxDB, err := n.storeServiceManager.StoreServiceByTMSId(tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get TTXDB store service for [%s]", tmsID)
	}

	// Create recovery manager with network adapter
	networkAdapter := &networkAdapter{network: n}

	// Create finality listener factory
	listenerFactory := &finalityListenerFactory{
		networkAdapter: networkAdapter,
		tmsProvider:    n.tmsProvider,
		tmsID:          tmsID,
		ttxDB:          ttxDB,
		tokensService:  tokensService,
		finalityTracer: n.finalityTracer,
	}

	manager := recovery.NewManager(
		logger,
		ttxDB,
		networkAdapter,
		tmsID.Namespace,
		listenerFactory,
		recoveryConfig,
	)

	// Start the recovery manager
	if err := manager.Start(); err != nil {
		return nil, errors.Wrapf(err, "failed to start recovery manager for [%s]", tmsID)
	}

	logger.Debugf("recovery manager started for namespace [%s]", tmsID.Namespace)

	return manager, nil
}

// networkAdapter adapts the Fabric Network to dep.Network interface
// It only implements AddFinalityListener which is what the recovery manager needs
type networkAdapter struct {
	network *Network
}

func (a *networkAdapter) AddFinalityListener(namespace string, txID string, listener network.FinalityListener) error {
	// Adapt network.FinalityListener to driver.FinalityListener
	driverListener := &finalityListenerAdapter{listener: listener}

	return a.network.AddFinalityListener(namespace, txID, driverListener)
}

func (a *networkAdapter) NewEnvelope() *network.Envelope {
	// This method is not used by the recovery manager, but required by dep.Network interface
	panic("NewEnvelope not implemented in networkAdapter")
}

func (a *networkAdapter) AnonymousIdentity() (view.Identity, error) {
	// This method is not used by the recovery manager, but required by dep.Network interface
	panic("AnonymousIdentity not implemented in networkAdapter")
}

func (a *networkAdapter) LocalMembership() *network.LocalMembership {
	// This method is not used by the recovery manager, but required by dep.Network interface
	panic("LocalMembership not implemented in networkAdapter")
}

func (a *networkAdapter) ComputeTxID(n *network.TxID) string {
	// This method is not used by the recovery manager, but required by dep.Network interface
	panic("ComputeTxID not implemented in networkAdapter")
}

// finalityListenerAdapter adapts network.FinalityListener to driver.FinalityListener
type finalityListenerAdapter struct {
	listener network.FinalityListener
}

func (a *finalityListenerAdapter) OnStatus(ctx context.Context, txID string, status int, message string, tokenRequestHash []byte) {
	a.listener.OnStatus(ctx, txID, status, message, tokenRequestHash)
}

func (a *finalityListenerAdapter) OnError(ctx context.Context, txID string, err error) {
	a.listener.OnError(ctx, txID, err)
}

type lookupListener struct {
	key   string
	wg    *sync.WaitGroup
	value []byte
	err   error
}

func (l *lookupListener) OnStatus(ctx context.Context, key string, value []byte) {
	logger.DebugfContext(ctx, "lookup transfer metadata key [%s], got value [%s][%v]", l.key, key, value)
	if l.key == key {
		l.value = value
		l.wg.Done()

		return
	}
}

func (l *lookupListener) OnError(ctx context.Context, key string, err error) {
	logger.DebugfContext(ctx, "lookup transfer metadata key [%s], got error [%s][%s]", l.key, key, err)
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

// NewSetupListenerProvider returns a provider for listeners that monitor public parameter updates.
func NewSetupListenerProvider(tmsProvider *token2.ManagementServiceProvider, tokensProvider *tokens.ServiceManager) *setupListenerProvider {
	return &setupListenerProvider{
		tmsProvider:    tmsProvider,
		tokensProvider: tokensProvider,
	}
}

type setupListenerProvider struct {
	tmsProvider    *token2.ManagementServiceProvider
	tokensProvider *tokens.ServiceManager
}

// GetListener returns a setupListener configured for the specified TMS ID.
func (p *setupListenerProvider) GetListener(tmsID token2.TMSID) lookup.Listener {
	return &setupListener{
		GetTMSProvider: func() *token2.ManagementServiceProvider { return p.tmsProvider },
		GetTokens: lazy.NewGetter[*tokens.Service](func() (*tokens.Service, error) {
			return p.tokensProvider.ServiceByTMSId(tmsID)
		}).Get,
		TMSID: tmsID,
	}
}

type setupListener struct {
	GetTMSProvider GetTMSProviderFunc
	GetTokens      GetTokensFunc
	TMSID          token2.TMSID
}

// OnStatus is triggered when the setup key (public parameters) is updated on the ledger.
func (s *setupListener) OnStatus(ctx context.Context, key string, value []byte) {
	logger.Infof("update TMS [%s] with key-value [%s][%s]", s.TMSID, key, utils.Hashable(value))
	tsmProvider := s.GetTMSProvider()
	if err := tsmProvider.Update(s.TMSID, value); err != nil {
		logger.Warnf("failed to update TMS [%s]: [%v]", key, err)
	}
	tokens, err := s.GetTokens()
	if err != nil {
		logger.Warnf("failed to get tokens db [%v]", err)

		return
	}
	if err := tokens.StorePublicParams(ctx, value); err != nil {
		logger.Warnf("failed to store public parameter key [%s]: [%v]", key, err)
	}
}

func (s *setupListener) OnError(ctx context.Context, key string, err error) {
	logger.Warnf("setup listener error for TMS [%s] key [%s]: [%v]", s.TMSID, key, err)
}
