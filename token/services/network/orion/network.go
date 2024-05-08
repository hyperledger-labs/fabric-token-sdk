/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package orion

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/keys"

	"github.com/hyperledger-labs/fabric-smart-client/platform/orion"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/events"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type NewVaultFunc = func(network, channel, namespace string) (vault.Vault, error)

type IdentityProvider interface {
	DefaultIdentity() view.Identity
}

type Network struct {
	sp          view2.ServiceProvider
	viewManager *view2.Manager
	tmsProvider *token2.ManagementServiceProvider
	n           *orion.NetworkService
	ip          IdentityProvider
	ledger      *ledger

	vaultCacheLock sync.RWMutex
	vaultCache     map[string]driver.Vault
	newVault       NewVaultFunc
	subscribers    *events.Subscribers
}

func NewNetwork(sp view2.ServiceProvider, ip IdentityProvider, n *orion.NetworkService, newVault NewVaultFunc) *Network {
	net := &Network{
		sp:          sp,
		ip:          ip,
		n:           n,
		viewManager: view2.GetManager(sp),
		tmsProvider: token2.GetManagementServiceProvider(sp),
		vaultCache:  map[string]driver.Vault{},
		newVault:    newVault,
		subscribers: events.NewSubscribers(),
	}
	net.ledger = &ledger{n: net}
	return net
}

func (n *Network) Name() string {
	return n.n.Name()
}

func (n *Network) Channel() string {
	return ""
}

func (n *Network) Vault(namespace string) (driver.Vault, error) {
	if len(namespace) == 0 {
		tms, err := n.tmsProvider.GetManagementService(token2.WithNetwork(n.n.Name()))
		if tms == nil || err != nil {
			return nil, errors.Errorf("empty namespace passed, cannot find TMS for [%s]: %v", n.n.Name(), err)
		}
		namespace = tms.Namespace()
	}

	// check cache
	n.vaultCacheLock.RLock()
	v, ok := n.vaultCache[namespace]
	n.vaultCacheLock.RUnlock()
	if ok {
		return v, nil
	}

	// lock
	n.vaultCacheLock.Lock()
	defer n.vaultCacheLock.Unlock()

	// check cache again
	v, ok = n.vaultCache[namespace]
	if ok {
		return v, nil
	}

	tokenVault, err := n.newVault(n.Name(), n.Channel(), namespace)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get token vault")
	}
	nv := &nv{
		v:          n.n.Vault(),
		tokenVault: tokenVault,
	}
	// store in cache
	n.vaultCache[namespace] = nv

	return nv, nil
}

func (n *Network) Broadcast(_ context.Context, blob interface{}) error {
	var err error
	switch b := blob.(type) {
	case driver.Envelope:
		_, err = n.viewManager.InitiateView(NewBroadcastView(n, b))
	default:
		_, err = n.viewManager.InitiateView(NewBroadcastView(n, b))
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
	return driver.TransientMap(tm), nil
}

func (n *Network) RequestApproval(context view.Context, tms *token2.ManagementService, requestRaw []byte, signer view.Identity, txID driver.TxID) (driver.Envelope, error) {
	envBoxed, err := view2.GetManager(context).InitiateView(NewRequestApprovalView(
		n, tms.Namespace(),
		requestRaw, signer, n.ComputeTxID(&txID),
	))
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
	pp, err := n.viewManager.InitiateView(NewPublicParamsRequestView(n.Name(), namespace))
	if err != nil {
		return nil, err
	}
	return pp.([]byte), nil
}

func (n *Network) QueryTokens(context view.Context, namespace string, IDs []*token.ID) ([][]byte, error) {
	resBoxed, err := view2.GetManager(context).InitiateView(NewRequestQueryTokensView(n, namespace, IDs))
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

	resBoxed, err := view2.GetManager(context).InitiateView(NewRequestSpentTokensView(n, namespace, sIDs))
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
		sp:          n.n.SP,
		namespace:   namespace,
		retryRunner: db.NewRetryRunner(-1, time.Second, true),
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

func (n *Network) LookupTransferMetadataKey(namespace string, startingTxID string, key string, timeout time.Duration) ([]byte, error) {
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
		),
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
	v          orion.Vault
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

func (v *nv) GetLastTxID() (string, error) {
	return v.v.GetLastTxID()
}

func (v *nv) Status(id string) (driver.ValidationCode, string, error) {
	vc, message, err := v.v.Status(id)
	return vc, message, err
}

func (v *nv) DiscardTx(id string, message string) error {
	return v.v.DiscardTx(id, message)
}

type ledger struct {
	n *Network
}

func (l *ledger) Status(id string) (driver.ValidationCode, error) {
	boxed, err := view2.GetManager(l.n.sp).InitiateView(NewRequestTxStatusView(l.n.Name(), "", id))
	if err != nil {
		return driver.Unknown, err
	}
	return boxed.(*TxStatusResponse).Status, nil
}

type FinalityListener struct {
	net         *Network
	root        driver.FinalityListener
	sp          token2.ServiceProvider
	network     string
	namespace   string
	retryRunner db.RetryRunner
}

func (t *FinalityListener) OnStatus(txID string, status int, message string) {
	if err := t.retryRunner.Run(func() error { return t.runOnStatus(txID, status, message) }); err != nil {
		logger.Errorf("failed running finality listener: %v", err)
	}
}

func (t *FinalityListener) runOnStatus(txID string, status int, message string) (err error) {
	defer func() { err = wrapRecover(recover()) }()
	boxed, err := view2.GetManager(t.sp).InitiateView(NewRequestTxStatusView(t.network, t.namespace, txID))
	if err != nil {
		return fmt.Errorf("failed retrieving token request [%s]: [%s]", txID, err)
	}
	t.root.OnStatus(txID, status, message, boxed.(*TxStatusResponse).TokenRequestReference)
	return
}

func wrapRecover(r any) error {
	if r != nil {
		return fmt.Errorf("panic caught: %v", r)
	}
	return nil
}
