/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package orion

import (
	"context"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/orion"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/events"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
	common2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/keys"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type NewVaultFunc = func(network, channel, namespace string) (vault.Vault, error)

type IdentityProvider interface {
	DefaultIdentity() view.Identity
}

type Network struct {
	viewManager    *view2.Manager
	tmsProvider    *token2.ManagementServiceProvider
	n              *orion.NetworkService
	ip             IdentityProvider
	ledger         *ledger
	nsFinder       common2.Configuration
	filterProvider common2.TransactionFilterProvider[*common2.AcceptTxInDBsFilter]

	vaultLazyCache      utils.LazyProvider[string, driver.Vault]
	tokenVaultLazyCache utils.LazyProvider[string, driver.TokenVault]
	subscribers         *events.Subscribers
	dbManager           *DBManager
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
		vaultLazyCache:      utils.NewLazyProvider(loader.loadVault),
		tokenVaultLazyCache: utils.NewLazyProvider(loader.loadTokenVault),
		subscribers:         events.NewSubscribers(), ledger: &ledger{network: n.Name(), viewManager: viewManager, dbManager: dbManager},
		dbManager: dbManager,
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

	// fetch public params and instantiate the tms
	ppRaw, err := n.FetchPublicParameters(ns)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to fetch public parameters for [%s]", tmsID)
	}
	return []token2.ServiceOption{token2.WithTMSID(tmsID), token2.WithPublicParameter(ppRaw)}, nil
}

func (n *Network) Vault(namespace string) (driver.Vault, error) {
	if len(namespace) == 0 {
		tms, err := n.tmsProvider.GetManagementService(token2.WithNetwork(n.n.Name()))
		if tms == nil || err != nil {
			return nil, errors.Errorf("empty namespace passed, cannot find TMS for [%s]: %v", n.n.Name(), err)
		}
		namespace = tms.Namespace()
	}

	return n.vaultLazyCache.Get(namespace)
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

func (n *Network) QueryTokens(context view.Context, namespace string, IDs []*token.ID) ([][]byte, error) {
	resBoxed, err := view2.GetManager(context).InitiateView(NewRequestQueryTokensView(n, namespace, IDs), context.Context())
	if err != nil {
		return nil, err
	}
	return resBoxed.([][]byte), nil
}

func (n *Network) AreTokensSpent(context view.Context, namespace string, tokenIDs []*token.ID, meta []string) ([]bool, error) {
	sIDs := make([]string, len(tokenIDs))
	var err error
	for i, id := range tokenIDs {
		sIDs[i], err = keys.CreateTokenKey(id.TxId, id.Index)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot compute spent id for [%v]", id)
		}
	}

	resBoxed, err := view2.GetManager(context).InitiateView(NewRequestSpentTokensView(n, namespace, sIDs), context.Context())
	if err != nil {
		return nil, err
	}
	return resBoxed.([]bool), nil
}

func (n *Network) LocalMembership() driver.LocalMembership {
	return &lm{
		lm: n.n.IdentityManager(),
		ip: n.ip,
	}
}

func (n *Network) AddFinalityListener(namespace string, txID string, listener driver.FinalityListener) error {
	wrapper := &FinalityListener{
		net:         n,
		root:        listener,
		network:     n.n.Name(),
		namespace:   namespace,
		retryRunner: db.NewRetryRunner(3, 100*time.Millisecond, true),
		viewManager: n.viewManager,
		dbManager:   n.dbManager,
	}
	n.subscribers.Set(txID, listener, wrapper)
	return n.n.Committer().AddFinalityListener(txID, wrapper)
}

func (n *Network) RemoveFinalityListener(txID string, listener driver.FinalityListener) error {
	wrapper, ok := n.subscribers.Get(txID, listener)
	if !ok {
		return errors.Errorf("listener was not registered")
	}
	return n.n.Committer().RemoveFinalityListener(txID, wrapper.(*FinalityListener))
}

func (n *Network) LookupTransferMetadataKey(namespace string, startingTxID string, key string, timeout time.Duration, _ bool) ([]byte, error) {
	k, err := keys.CreateTransferActionMetadataKey(key)
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

type nv struct {
	v orion.Vault
}

func (v *nv) GetLastTxID() (string, error) {
	return v.v.GetLastTxID()
}

func (v *nv) NewQueryExecutor() (driver.QueryExecutor, error) {
	panic("not supported")
}

func (v *nv) Status(id string) (driver.ValidationCode, string, error) {
	return v.v.Status(id)
}

func (v *nv) DiscardTx(id string, message string) error {
	return v.v.DiscardTx(id, message)
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
	network     string
	viewManager *view2.Manager
	dbManager   *DBManager
}

func (l *ledger) Status(id string) (driver.ValidationCode, error) {
	boxed, err := l.viewManager.InitiateView(NewRequestTxStatusView(l.network, "", id, l.dbManager), context.TODO())
	if err != nil {
		return driver.Unknown, err
	}
	return boxed.(*TxStatusResponse).Status, nil
}

type FinalityListener struct {
	net         *Network
	root        driver.FinalityListener
	network     string
	namespace   string
	retryRunner db.RetryRunner
	viewManager *view2.Manager
	dbManager   *DBManager
}

func (t *FinalityListener) OnStatus(_ context.Context, txID string, status int, message string) {
	if err := t.retryRunner.Run(func() error { return t.runOnStatus(txID, status, message) }); err != nil {
		logger.Errorf("failed running finality listener: %v", err)
	}
}

func (t *FinalityListener) runOnStatus(txID string, status int, message string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.Errorf("panic caught: %v", r)
		}
	}()
	boxed, err := t.viewManager.InitiateView(NewRequestTxStatusView(t.network, t.namespace, txID, t.dbManager), context.TODO())
	if err != nil {
		return errors.Wrapf(err, "failed retrieving token request [%s]", txID)
	}
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

func (l *loader) loadVault(namespace string) (driver.Vault, error) {
	return &nv{v: l.vault}, nil
}

func (l *loader) loadTokenVault(namespace string) (driver.TokenVault, error) {
	tv, err := l.newVault(l.name, l.channel, namespace)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get token vault")
	}
	return &tokenVault{tokenVault: tv}, nil
}
