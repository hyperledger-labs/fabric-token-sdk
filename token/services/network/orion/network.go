/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package orion

import (
	"context"
	"sync"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/orion"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	api2 "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/rws/keys"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type NewVaultFunc = func(network, channel, namespace string) (vault.TokenVault, error)

type IdentityProvider interface {
	DefaultIdentity() view.Identity
}

type Network struct {
	sp     view2.ServiceProvider
	n      *orion.NetworkService
	ip     IdentityProvider
	ledger *ledger

	vaultCacheLock sync.RWMutex
	vaultCache     map[string]driver.Vault
	newVault       NewVaultFunc
}

func NewNetwork(sp view2.ServiceProvider, ip IdentityProvider, n *orion.NetworkService, newVault NewVaultFunc) *Network {
	net := &Network{
		sp:         sp,
		ip:         ip,
		n:          n,
		vaultCache: map[string]driver.Vault{},
		newVault:   newVault,
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
		tms := token2.GetManagementService(n.sp, token2.WithNetwork(n.n.Name()))
		if tms == nil {
			return nil, errors.Errorf("empty namespace passed, cannot find TMS for [%s]", n.n.Name())
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

func (n *Network) StoreEnvelope(env driver.Envelope) error {
	rws, err := n.n.Vault().GetRWSet(env.TxID(), env.Results())
	if err != nil {
		return errors.WithMessagef(err, "failed to get rwset")
	}
	rws.Done()

	rawEnv, err := env.Bytes()
	if err != nil {
		return errors.WithMessagef(err, "failed marshalling tx env [%s]", env.TxID())
	}

	return n.n.Vault().StoreEnvelope(env.TxID(), rawEnv)
}

func (n *Network) EnvelopeExists(id string) bool {
	return n.n.EnvelopeService().Exists(id)
}

func (n *Network) Broadcast(_ context.Context, blob interface{}) error {
	var err error
	switch b := blob.(type) {
	case driver.Envelope:
		_, err = view2.GetManager(n.sp).InitiateView(NewBroadcastView(n, b))
	default:
		_, err = view2.GetManager(n.sp).InitiateView(NewBroadcastView(n, b))
	}
	return err
}

func (n *Network) IsFinalForParties(id string, endpoints ...view.Identity) error {
	panic("implement me")
}

func (n *Network) IsFinal(ctx context.Context, id string) error {
	return n.n.Finality().IsFinal(ctx, id)
}

func (n *Network) NewEnvelope() driver.Envelope {
	return n.n.TransactionManager().NewEnvelope()
}

func (n *Network) StoreTransient(id string, transient driver.TransientMap) error {
	return n.n.Vault().StoreTransient(id, orion.TransientMap(transient))
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
	pp, err := view2.GetManager(n.sp).InitiateView(NewPublicParamsRequestView(n.Name(), namespace))
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

func (n *Network) SubscribeTxStatusChanges(txID string, listener driver.TxStatusChangeListener) error {
	return n.n.Committer().SubscribeTxStatusChanges(txID, listener)
}

func (n *Network) UnsubscribeTxStatusChanges(txID string, listener driver.TxStatusChangeListener) error {
	return n.n.Committer().UnsubscribeTxStatusChanges(txID, listener)
}

func (n *Network) LookupTransferMetadataKey(namespace string, startingTxID string, key string, timeout time.Duration) ([]byte, error) {
	k, err := keys.CreateTransferActionMetadataKey(key)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to generate transfer action metadata key from [%s]", key)
	}
	pp, err := view2.GetManager(n.sp).InitiateView(
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
	v          *orion.Vault
	tokenVault driver.TokenVault
}

func (v *nv) QueryEngine() api2.QueryEngine {
	return v.tokenVault.QueryEngine()
}

func (v *nv) CertificationStorage() api2.CertificationStorage {
	return v.tokenVault.CertificationStorage()
}

func (v *nv) DeleteTokens(ns string, ids ...*token.ID) error {
	return v.tokenVault.DeleteTokens(ns, ids...)
}

func (v *nv) GetLastTxID() (string, error) {
	return v.v.GetLastTxID()
}

func (v *nv) Exists(id *token.ID) bool {
	return v.tokenVault.CertificationStorage().Exists(id)
}

func (v *nv) Store(certifications map[*token.ID][]byte) error {
	return v.tokenVault.CertificationStorage().Store(certifications)
}

func (v *nv) Status(txID string) (driver.ValidationCode, error) {
	vc, err := v.v.Status(txID)
	return driver.ValidationCode(vc), err
}

func (v *nv) DiscardTx(txID string) error {
	return v.v.DiscardTx(txID)
}

type ledger struct {
	n *Network
}

func (l *ledger) Status(id string) (driver.ValidationCode, error) {
	boxed, err := view2.GetManager(l.n.sp).InitiateView(NewRequestTxStatusView(l.n, id))
	if err != nil {
		return driver.Unknown, err
	}
	return boxed.(driver.ValidationCode), nil
}
