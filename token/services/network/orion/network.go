/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package orion

import (
	"context"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/lazy"
	"github.com/hyperledger-labs/fabric-smart-client/platform/orion"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracing"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/common"
	common2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/translator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
)

type NewVaultFunc = func(network, channel, namespace string) (driver.TokenVault, error)

type IdentityProvider interface {
	DefaultIdentity() view.Identity
}

type Network struct {
	viewManager             *view2.Manager
	tmsProvider             *token2.ManagementServiceProvider
	n                       *orion.NetworkService
	ip                      IdentityProvider
	ledger                  *ledger
	nsFinder                common2.Configuration
	filterProvider          common2.TransactionFilterProvider[*common2.AcceptTxInDBsFilter]
	finalityTracer          trace.Tracer
	tokenQueryExecutor      driver.TokenQueryExecutor
	spentTokenQueryExecutor driver.SpentTokenQueryExecutor

	tokenVaultLazyCache lazy.Provider[string, driver.TokenVault]
	dbManager           *DBManager
	flm                 FinalityListenerManager
	keyTranslator       translator.KeyTranslator
}

func NewNetwork(
	viewManager *view2.Manager,
	tmsProvider *token2.ManagementServiceProvider,
	ip IdentityProvider,
	n *orion.NetworkService,
	newVault NewVaultFunc,
	nsFinder common2.Configuration,
	filterProvider common2.TransactionFilterProvider[*common2.AcceptTxInDBsFilter],
	dbManager *DBManager,
	flm FinalityListenerManager,
	tokenQueryExecutor driver.TokenQueryExecutor,
	spentTokenQueryExecutor driver.SpentTokenQueryExecutor,
	tracerProvider trace.TracerProvider,
	keyTranslator translator.KeyTranslator,
) *Network {
	loader := &loader{
		newVault: newVault,
		name:     n.Name(),
		channel:  "",
		vault:    n.Vault(),
	}
	return &Network{
		nsFinder:            nsFinder,
		filterProvider:      filterProvider,
		ip:                  ip,
		n:                   n,
		viewManager:         viewManager,
		tmsProvider:         tmsProvider,
		tokenVaultLazyCache: lazy.NewProvider(loader.loadTokenVault),
		ledger:              &ledger{network: n.Name(), viewManager: viewManager, dbManager: dbManager},
		finalityTracer: tracerProvider.Tracer("finality_listener", tracing.WithMetricsOpts(tracing.MetricsOpts{
			Namespace:  "tokensdk_orion",
			LabelNames: []tracing.LabelName{},
		})),
		tokenQueryExecutor:      tokenQueryExecutor,
		spentTokenQueryExecutor: spentTokenQueryExecutor,
		dbManager:               dbManager,
		flm:                     flm,
		keyTranslator:           keyTranslator,
	}
}

func (n *Network) Name() string {
	return n.n.Name()
}

func (n *Network) Channel() string {
	return ""
}

func (n *Network) Normalize(opt *token2.ServiceOptions) (*token2.ServiceOptions, error) {
	if len(opt.Network) == 0 {
		opt.Network = n.n.Name()
	}
	if opt.Network != n.n.Name() {
		return nil, errors.Errorf("invalid network [%s], expected [%s]", opt.Network, n.n.Name())
	}

	if len(opt.Channel) != 0 {
		return nil, errors.Errorf("invalid channel [%s], expected []", opt.Channel)
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
		Network:   n.Name(),
		Namespace: ns,
	}
	transactionFilter, err := n.filterProvider.New(tmsID)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to create transaction filter for [%s]", tmsID)
	}
	if err := n.n.Committer().AddTransactionFilter(transactionFilter); err != nil {
		return nil, errors.WithMessagef(err, "failed to fetch attach transaction filter [%s]", tmsID)
	}
	return nil, nil
}

func (n *Network) TokenVault(namespace string) (driver.TokenVault, error) {
	if len(namespace) == 0 {
		tms, err := n.tmsProvider.GetManagementService(token2.WithNetwork(n.n.Name()))
		if tms == nil || err != nil {
			return nil, errors.Errorf("empty namespace passed, cannot find TMS for [%s]: %v", n.n.Name(), err)
		}
		namespace = tms.Namespace()
	}

	return n.tokenVaultLazyCache.Get(namespace)
}

func (n *Network) Broadcast(ctx context.Context, blob interface{}) error {
	var err error
	switch b := blob.(type) {
	case driver.Envelope:
		_, err = n.viewManager.InitiateView(NewBroadcastView(n.dbManager, n.Name(), b), ctx)
	default:
		_, err = n.viewManager.InitiateView(NewBroadcastView(n.dbManager, n.Name(), b), ctx)
	}
	return err
}

func (n *Network) NewEnvelope() driver.Envelope {
	return n.n.TransactionManager().NewEnvelope()
}

func (n *Network) StoreTransient(id string, transient driver.TransientMap) error {
	return n.n.Vault().StoreTransient(id, transient)
}

func (n *Network) TransientExists(id string) bool {
	return n.n.MetadataService().Exists(id)
}

func (n *Network) GetTransient(id string) (driver.TransientMap, error) {
	tm, err := n.n.MetadataService().LoadTransient(id)
	if err != nil {
		return nil, err
	}
	return tm, nil
}

func (n *Network) RequestApproval(context view.Context, tms *token2.ManagementService, requestRaw []byte, signer view.Identity, txID driver.TxID) (driver.Envelope, error) {
	envBoxed, err := view2.GetManager(context).InitiateView(NewRequestApprovalView(
		n.dbManager,
		n.n.Name(), tms.Namespace(),
		requestRaw, signer, n.ComputeTxID(&txID),
	), context.Context())
	if err != nil {
		return nil, err
	}
	return envBoxed.(driver.Envelope), nil
}

func (n *Network) ComputeTxID(id *driver.TxID) string {
	logger.Debugf("compute tx id for [%s]", id.String())
	temp := &orion.TxID{
		Nonce:   id.Nonce,
		Creator: id.Creator,
	}
	res := n.n.TransactionManager().ComputeTxID(temp)
	id.Nonce = temp.Nonce
	id.Creator = temp.Creator
	return res
}

func (n *Network) FetchPublicParameters(namespace string) ([]byte, error) {
	pp, err := n.viewManager.InitiateView(NewPublicParamsRequestView(n.Name(), namespace), context.TODO())
	if err != nil {
		return nil, err
	}
	return pp.([]byte), nil
}

func (n *Network) QueryTokens(context context.Context, namespace string, IDs []*token.ID) ([][]byte, error) {
	return n.tokenQueryExecutor.QueryTokens(context, namespace, IDs)
}

func (n *Network) AreTokensSpent(context context.Context, namespace string, tokenIDs []*token.ID, meta []string) ([]bool, error) {
	return n.spentTokenQueryExecutor.QuerySpentTokens(context, namespace, tokenIDs, meta)
}

func (n *Network) LocalMembership() driver.LocalMembership {
	return &lm{
		lm: n.n.IdentityManager(),
		ip: n.ip,
	}
}

func (n *Network) AddFinalityListener(namespace string, txID string, listener driver.FinalityListener) error {
	return n.flm.AddFinalityListener(namespace, txID, listener)
}

func (n *Network) RemoveFinalityListener(txID string, listener driver.FinalityListener) error {
	return n.flm.RemoveFinalityListener(txID, listener)
}

func (n *Network) LookupTransferMetadataKey(namespace string, startingTxID string, key string, timeout time.Duration, _ bool) ([]byte, error) {
	k, err := n.keyTranslator.CreateTransferActionMetadataKey(key)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to generate transfer action metadata key from [%s]", key)
	}
	pp, err := n.viewManager.InitiateView(
		NewLookupKeyRequestView(
			n.Name(),
			namespace,
			startingTxID,
			orionKey(k),
			timeout,
		), context.TODO(),
	)
	if err != nil {
		return nil, err
	}
	return pp.([]byte), nil
}

func (n *Network) Ledger() (driver.Ledger, error) {
	return n.ledger, nil
}

func (n *Network) ProcessNamespace(namespace string) error {
	// Not supported
	return nil
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
	network     string
	viewManager *view2.Manager
	dbManager   *DBManager
}

func (l *ledger) Status(id string) (driver.ValidationCode, error) {
	boxed, err := l.viewManager.InitiateView(NewRequestTxStatusView(l.network, "", id, l.dbManager), context.TODO())
	if err != nil {
		return driver.Unknown, errors.Errorf("failed to get status for [%s]", id)
	}
	return boxed.(*TxStatusResponse).Status, nil
}

type FinalityListener struct {
	root        driver.FinalityListener
	network     string
	namespace   string
	tracer      trace.Tracer
	retryRunner common.RetryRunner
	viewManager *view2.Manager
	dbManager   *DBManager
}

func (t *FinalityListener) OnStatus(ctx context.Context, txID string, status int, message string) {
	newCtx, span := t.tracer.Start(ctx, "on_status")
	defer span.End()
	if err := t.retryRunner.Run(func() error { return t.runOnStatus(newCtx, txID, status, message) }); err != nil {
		logger.Errorf("failed running finality listener: %v", err)
	}
}

func (t *FinalityListener) runOnStatus(ctx context.Context, txID string, status int, message string) (err error) {
	span := trace.SpanFromContext(ctx)
	span.AddEvent("start_run_on_status")
	defer span.AddEvent("end_run_on_status")

	defer func() {
		if r := recover(); r != nil {
			err = errors.Errorf("panic caught: %v", r)
		}
	}()
	span.AddEvent("request_tx_status_view")
	boxed, err := t.viewManager.InitiateView(NewRequestTxStatusView(t.network, t.namespace, txID, t.dbManager), ctx)
	if err != nil {
		return errors.Wrapf(err, "failed retrieving token request [%s]", txID)
	}
	span.AddEvent("received_tx_status")
	statusResponse, ok := boxed.(*TxStatusResponse)
	if !ok {
		return errors.Errorf("failed retrieving token request, expected TxStatusResponse [%s]", txID)
	}
	if statusResponse == nil {
		return errors.Errorf("expected status response to be non-nil for [%s]", txID)
	}
	if statusResponse.Status != status {
		return errors.Errorf("expected status [%v], got [%v]", status, statusResponse.Status)
	}
	if statusResponse.Status == driver.Valid && len(statusResponse.TokenRequestReference) == 0 {
		return errors.Errorf("expected status response to be non-nil for a valid transaction")
	}

	t.root.OnStatus(
		ctx,
		txID,
		status,
		message,
		boxed.(*TxStatusResponse).TokenRequestReference,
	)
	return nil
}

type loader struct {
	newVault NewVaultFunc
	name     string
	channel  string
	vault    orion.Vault
}

func (l *loader) loadTokenVault(namespace string) (driver.TokenVault, error) {
	tv, err := l.newVault(l.name, l.channel, namespace)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get token vault")
	}
	return &tokenVault{tokenVault: tv}, nil
}
