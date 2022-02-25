/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package network

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type GetFunc func() (view.Identity, []byte, error)

type TxID struct {
	Nonce   []byte
	Creator []byte
}

func (t *TxID) String() string {
	return fmt.Sprintf("[%s:%s]", base64.StdEncoding.EncodeToString(t.Nonce), base64.StdEncoding.EncodeToString(t.Creator))
}

type TransientMap map[string][]byte

func (m TransientMap) Set(key string, raw []byte) error {
	m[key] = raw

	return nil
}

func (m TransientMap) Get(id string) []byte {
	return m[id]
}

func (m TransientMap) IsEmpty() bool {
	return len(m) == 0
}

func (m TransientMap) Exists(key string) bool {
	_, ok := m[key]
	return ok
}

func (m TransientMap) SetState(key string, state interface{}) error {
	raw, err := json.Marshal(state)
	if err != nil {
		return err
	}
	m[key] = raw

	return nil
}

func (m TransientMap) GetState(key string, state interface{}) error {
	value, ok := m[key]
	if !ok {
		return errors.Errorf("transient map key [%s] does not exists", key)
	}
	if len(value) == 0 {
		return errors.Errorf("transient map key [%s] is empty", key)
	}

	return json.Unmarshal(value, state)
}

type Envelope struct {
	e driver.Envelope
}

func (e *Envelope) Results() []byte {
	return e.e.Results()
}

func (e *Envelope) Bytes() ([]byte, error) {
	return e.e.Bytes()
}

func (e *Envelope) FromBytes(raw []byte) error {
	return e.e.FromBytes(raw)
}

func (e *Envelope) TxID() string {
	return e.e.TxID()
}

func (e *Envelope) Nonce() []byte {
	return e.e.Nonce()
}

func (e *Envelope) Creator() []byte {
	return e.e.Creator()
}

func (e *Envelope) MarshalJSON() ([]byte, error) {
	raw, err := e.e.Bytes()
	if err != nil {
		return nil, err
	}
	return json.Marshal(raw)
}

func (e *Envelope) UnmarshalJSON(raw []byte) error {
	var r []byte
	err := json.Unmarshal(raw, &r)
	if err != nil {
		return err
	}
	return e.e.FromBytes(r)
}

type RWSet struct {
	rws driver.RWSet
}

func (s *RWSet) Done() {
	s.rws.Done()
}

type Vault struct {
	v driver.Vault
}

func (v *Vault) GetLastTxID() (string, error) {
	return v.v.GetLastTxID()
}

func (v *Vault) ListUnspentTokens() (*token2.UnspentTokens, error) {
	return v.v.ListUnspentTokens()
}

func (v *Vault) Exists(id *token2.ID) bool {
	return v.v.Exists(id)
}

func (v *Vault) Store(certifications map[*token2.ID][]byte) error {
	return v.v.Store(certifications)
}

func (v *Vault) TokenVault() *vault.Vault {
	return v.v.TokenVault()
}

type LocalMembership struct {
	lm driver.LocalMembership
}

func (l *LocalMembership) DefaultIdentity() view.Identity {
	return l.lm.DefaultIdentity()
}

func (l *LocalMembership) AnonymousIdentity() view.Identity {
	return l.lm.AnonymousIdentity()
}

func (l *LocalMembership) IsMe(id view.Identity) bool {
	return l.lm.IsMe(id)
}

func (l *LocalMembership) GetLongTermIdentity(label string) (string, string, view.Identity, error) {
	return l.lm.GetLongTermIdentity(label)
}

func (l *LocalMembership) GetLongTermIdentifier(id view.Identity) (string, error) {
	return l.lm.GetLongTermIdentifier(id)
}

func (l *LocalMembership) GetAnonymousIdentity(label string, auditInfo []byte) (string, string, GetFunc, error) {
	id, eID, getFunc, err := l.lm.GetAnonymousIdentity(label, auditInfo)
	if err != nil {
		return "", "", nil, err
	}
	return id, eID, GetFunc(getFunc), nil
}

func (l *LocalMembership) GetAnonymousIdentifier(label string) (string, error) {
	return l.lm.GetAnonymousIdentifier(label)
}

func (l *LocalMembership) RegisterIdentity(id string, typ string, path string) error {
	return l.lm.RegisterIdentity(id, typ, path)
}

type Network struct {
	n driver.Network
}

func (n *Network) Name() string {
	return n.n.Name()
}

func (n *Network) Channel() string {
	return n.n.Channel()
}

func (n *Network) Vault(namespace string) (*Vault, error) {
	v, err := n.n.Vault(namespace)
	if err != nil {
		return nil, err
	}
	return &Vault{v: v}, nil
}

func (n *Network) GetRWSet(id string, results []byte) (*RWSet, error) {
	rws, err := n.n.GetRWSet(id, results)
	if err != nil {
		return nil, err
	}
	return &RWSet{rws: rws}, nil
}

func (n *Network) StoreEnvelope(id string, env []byte) error {
	return n.n.StoreEnvelope(id, env)
}

func (n *Network) Broadcast(blob interface{}) error {
	switch b := blob.(type) {
	case *Envelope:
		return n.n.Broadcast(b.e)
	default:
		return n.n.Broadcast(b)
	}
}

func (n *Network) IsFinalForParties(id string, endpoints ...view.Identity) error {
	return n.n.IsFinalForParties(id, endpoints...)
}

func (n *Network) IsFinal(id string) error {
	return n.n.IsFinal(id)
}

func (n *Network) AnonymousIdentity() view.Identity {
	return n.n.LocalMembership().AnonymousIdentity()
}

func (n *Network) NewEnvelope() *Envelope {
	return &Envelope{e: n.n.NewEnvelope()}
}

func (n *Network) StoreTransient(id string, transient TransientMap) error {
	return n.n.StoreTransient(id, driver.TransientMap(transient))
}

func (n *Network) RequestApproval(context view.Context, namespace string, requestRaw []byte, signer view.Identity, txID TxID) (*Envelope, error) {
	env, err := n.n.RequestApproval(context, namespace, requestRaw, signer, driver.TxID{
		Nonce:   txID.Nonce,
		Creator: txID.Creator,
	})
	if err != nil {
		return nil, err
	}
	return &Envelope{e: env}, nil
}

func (n *Network) ComputeTxID(id *TxID) string {
	temp := &driver.TxID{
		Nonce:   id.Nonce,
		Creator: id.Creator,
	}
	txID := n.n.ComputeTxID(temp)
	id.Nonce = temp.Nonce
	id.Creator = temp.Creator
	return txID
}

func (n *Network) FetchPublicParameters(namespace string) ([]byte, error) {
	return n.n.FetchPublicParameters(namespace)
}

func (n *Network) QueryTokens(context view.Context, namespace string, IDs []*token2.ID) ([][]byte, error) {
	return n.n.QueryTokens(context, namespace, IDs)
}

func (n *Network) LocalMembership() *LocalMembership {
	return &LocalMembership{lm: n.n.LocalMembership()}
}

func (n *Network) GetEnrollmentID(raw []byte) (string, error) {
	return n.n.GetEnrollmentID(raw)
}

type Provider struct {
	sp view2.ServiceProvider

	lock     sync.Mutex
	networks map[string]*Network
}

func NewProvider(sp view2.ServiceProvider) *Provider {
	ms := &Provider{
		sp:       sp,
		networks: map[string]*Network{},
	}
	return ms
}

func (np *Provider) GetNetwork(network string, channel string) (*Network, error) {
	np.lock.Lock()
	defer np.lock.Unlock()

	logger.Debugf("GetNetwork: [%s:%s]", network, channel)

	key := network + channel
	service, ok := np.networks[key]
	if !ok {
		var err error
		service, err = np.newNetwork(network, channel)
		if err != nil {
			logger.Errorf("Failed to get network: [%s:%s] %s", network, channel, err)
			return nil, err
		}
		np.networks[key] = service
	}
	return service, nil
}

func (np *Provider) newNetwork(network string, channel string) (*Network, error) {
	for _, d := range drivers {
		nw, err := d.New(np.sp, network, channel)
		if err != nil {
			logger.Errorf("Failed to create network [%s:%s]: %s", network, channel, err)
		}
		if nw != nil {
			return &Network{n: nw}, nil
		}
	}
	return nil, errors.Errorf("no network driver found for [%s:%s]", network, channel)
}

func GetInstance(sp view2.ServiceProvider, network, channel string) *Network {
	s, err := sp.GetService(&Provider{})
	if err != nil {
		panic(fmt.Sprintf("Failed to get service: %s", err))
	}
	n, err := s.(*Provider).GetNetwork(network, channel)
	if err != nil {
		logger.Errorf("Failed to get network [%s:%s]: %s", network, channel, err)
		return nil
	}
	return n
}
