/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package orion

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/orion"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"sync"
)

type Network struct {
	sp view2.ServiceProvider
	n  *orion.NetworkService

	vaultCacheLock sync.RWMutex
	vaultCache     map[string]driver.Vault
}

func NewNetwork(sp view2.ServiceProvider, n *orion.NetworkService) *Network {
	return &Network{sp: sp, n: n}
}

func (n *Network) Name() string {
	return n.n.Name()
}

func (n *Network) Channel() string {
	return ""
}

func (n *Network) Vault(namespace string) (driver.Vault, error) {
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

	tokenVault := vault.New(n.sp, n.Channel(), namespace, NewVault(n.n))
	nv := &nv{
		v:          n.n.Vault(),
		tokenVault: tokenVault,
	}
	// store in cache
	n.vaultCache[namespace] = nv

	return nv, nil
}

func (n *Network) GetRWSet(id string, results []byte) (driver.RWSet, error) {
	rws, err := n.n.Vault().GetRWSet(id, results)
	if err != nil {
		return nil, err
	}
	return rws, nil
}

func (n *Network) StoreEnvelope(id string, env []byte) error {
	return n.n.Vault().StoreEnvelope(id, env)
}

func (n *Network) Broadcast(blob interface{}) error {
	panic("implement me")
}

func (n *Network) IsFinalForParties(id string, endpoints ...view.Identity) error {
	panic("implement me")
}

func (n *Network) IsFinal(id string) error {
	panic("implement me")
}

func (n *Network) NewEnvelope() driver.Envelope {
	return n.n.TransactionManager().NewEnvelope()
}

func (n *Network) StoreTransient(id string, transient driver.TransientMap) error {
	return n.n.Vault().StoreTransient(id, orion.TransientMap(transient))
}

func (n *Network) RequestApproval(context view.Context, namespace string, requestRaw []byte, signer view.Identity, txID driver.TxID) (driver.Envelope, error) {
	envBoxed, err := view2.GetManager(context).InitiateView(NewRequestApprovalView(
		n, namespace,
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

func (n *Network) QueryTokens(context view.Context, namespace string, IDs []*token2.ID) ([][]byte, error) {
	panic("implement me")
}

func (n *Network) LocalMembership() driver.LocalMembership {
	panic("implement me")
}

func (n *Network) GetEnrollmentID(raw []byte) (string, error) {
	panic("implement me")
}

type nv struct {
	v          *orion.Vault
	tokenVault *vault.Vault
}

func (v *nv) GetLastTxID() (string, error) {
	return v.v.GetLastTxID()
}

func (v *nv) ListUnspentTokens() (*token2.UnspentTokens, error) {
	return v.tokenVault.QueryEngine().ListUnspentTokens()
}

func (v *nv) Exists(id *token2.ID) bool {
	return v.tokenVault.CertificationStorage().Exists(id)
}

func (v *nv) Store(certifications map[*token2.ID][]byte) error {
	return v.tokenVault.CertificationStorage().Store(certifications)
}

func (v *nv) TokenVault() *vault.Vault {
	return v.tokenVault
}
