/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package network

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/services/chaincode"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/services/fpc"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	fabric3 "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

const (
	InvokeFunction            = "invoke"
	AddIssuerFunction         = "addIssuer"
	QueryPublicParamsFunction = "queryPublicParams"
	AddAuditorFunction        = "addAuditor"
	AddCertifierFunction      = "addCertifier"
	QueryTokensFunctions      = "queryTokens"
)

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
	e *fabric.Envelope
}

func (e *Envelope) Results() []byte {
	return e.e.Results()
}

func (e *Envelope) Bytes() ([]byte, error) {
	return e.e.Bytes()
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
	rws *fabric.RWSet
}

func (s *RWSet) Done() {
	s.rws.Done()
}

type Vault struct {
	v          *fabric.Vault
	tokenVault *vault.Vault
}

func (v *Vault) GetLastTxID() (string, error) {
	return v.v.GetLastTxID()
}

func (v *Vault) ListUnspentTokens() (*token2.UnspentTokens, error) {
	return v.tokenVault.QueryEngine().ListUnspentTokens()
}

func (v *Vault) Exists(id *token2.ID) bool {
	return v.tokenVault.CertificationStorage().Exists(id)
}

func (v *Vault) Store(certifications map[*token2.ID][]byte) error {
	return v.tokenVault.CertificationStorage().Store(certifications)
}

func (v *Vault) TokenVault() *vault.Vault {
	return v.tokenVault
}

type LocalMembership struct {
	lm *fabric.LocalMembership
}

func (l *LocalMembership) DefaultIdentity() view.Identity {
	return l.lm.DefaultIdentity()
}

func (l *LocalMembership) IsMe(id view.Identity) bool {
	return l.lm.IsMe(id)
}

func (l *LocalMembership) GetIdentityInfoByLabel(mspType string, label string) *fabric.IdentityInfo {
	return l.lm.GetIdentityInfoByLabel(mspType, label)
}

func (l *LocalMembership) GetIdentityInfoByIdentity(mspType string, id view.Identity) *fabric.IdentityInfo {
	return l.lm.GetIdentityInfoByIdentity(mspType, id)
}

type Network struct {
	n  *fabric.NetworkService
	ch *fabric.Channel
	sp view2.ServiceProvider
}

func (n *Network) Name() string {
	return n.n.Name()
}

func (n *Network) Channel() string {
	return n.ch.Name()
}

func (n *Network) Vault(namespace string) (*Vault, error) {
	tokenVault := vault.New(n.sp, n.Channel(), namespace, fabric3.NewVault(n.ch))
	return &Vault{
		v:          n.ch.Vault(),
		tokenVault: tokenVault,
	}, nil
}

func (n *Network) GetRWSet(id string, results []byte) (*RWSet, error) {
	rws, err := n.ch.Vault().GetRWSet(id, results)
	if err != nil {
		return nil, err
	}
	return &RWSet{rws: rws}, nil
}

func (n *Network) StoreEnvelope(id string, env []byte) error {
	return n.ch.Vault().StoreEnvelope(id, env)
}

func (n *Network) Broadcast(blob interface{}) error {
	switch b := blob.(type) {
	case *Envelope:
		return n.n.Ordering().Broadcast(b.e)
	default:
		return n.n.Ordering().Broadcast(b)
	}
}

func (n *Network) IsFinalForParties(id string, endpoints ...view.Identity) error {
	return n.ch.Finality().IsFinalForParties(id, endpoints...)
}

func (n *Network) IsFinal(id string) error {
	return n.ch.Finality().IsFinal(id)
}

func (n *Network) AnonymousIdentity() view.Identity {
	return n.n.LocalMembership().AnonymousIdentity()
}

func (n *Network) NewEnvelope() *Envelope {
	return &Envelope{e: n.n.TransactionManager().NewEnvelope()}
}

func (n *Network) StoreTransient(id string, transient TransientMap) error {
	return n.ch.Vault().StoreTransient(id, fabric.TransientMap(transient))
}

func (n *Network) RequestApproval(context view.Context, namespace string, requestRaw []byte, signer view.Identity, txID TxID) (*Envelope, error) {
	if exist, err := fpc.GetChannel(context, n.n.Name(), n.ch.Name()).EnclaveRegistry().IsPrivate(namespace); err == nil && exist {
		// TODO: FPC does not support transient yet
		env, err := chaincode.NewEndorseView(
			namespace,
			InvokeFunction,
			requestRaw,
		).WithNetwork(
			n.n.Name(),
		).WithChannel(
			n.ch.Name(),
		).WithSignerIdentity(
			signer,
		).WithTxID(
			fabric.TxID{
				Nonce:   txID.Nonce,
				Creator: txID.Creator,
			},
		).Endorse(context)
		if err != nil {
			return nil, err
		}
		return &Envelope{e: env}, nil
	} else {
		env, err := chaincode.NewEndorseView(
			namespace,
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
		return &Envelope{e: env}, nil
	}
}

func (n *Network) ComputeTxID(id *TxID) string {
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

func (n *Network) AddIssuer(context view.Context, pk []byte) error {
	ts := token.GetManagementService(
		context,
		token.WithNetwork(n.Name()),
		token.WithChannel(n.Channel()),
	)

	_, err := context.RunView(
		chaincode.NewInvokeView(
			ts.Namespace(),
			AddIssuerFunction,
			pk,
		).WithNetwork(
			n.Name(),
		).WithChannel(
			n.Channel(),
		),
	)
	return err
}

func (n *Network) FetchPublicParameters(namespace string) ([]byte, error) {
	ppBoxed, err := view2.GetManager(n.sp).InitiateView(
		chaincode.NewQueryView(
			namespace,
			QueryPublicParamsFunction,
		).WithNetwork(n.Name()).WithChannel(n.Channel()),
	)
	if err != nil {
		return nil, err
	}
	return ppBoxed.([]byte), nil
}

func (n *Network) RegisterAuditor(context view.Context, namespace string, id view.Identity) error {
	_, err := context.RunView(chaincode.NewInvokeView(
		namespace,
		AddAuditorFunction,
		id.Bytes(),
	).WithNetwork(n.Name()).WithChannel(n.Channel()))
	return err
}

func (n *Network) RegisterCertifier(context view.Context, namespace string, id view.Identity) error {
	_, err := context.RunView(chaincode.NewInvokeView(
		namespace,
		AddCertifierFunction,
		id.Bytes(),
	).WithNetwork(n.Name()).WithChannel(n.Channel()).WithSignerIdentity(
		fabric.GetFabricNetworkService(context, n.Name()).IdentityProvider().DefaultIdentity(),
	))
	return err
}

func (n *Network) QueryTokens(context view.Context, namespace string, IDs []*token2.ID) ([][]byte, error) {
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
		return nil, errors.WithMessagef(err, "failed quering tokens")
	}

	// Unbox
	raw, ok := payloadBoxed.([]byte)
	if !ok {
		return nil, errors.Errorf("expected []byte from TCC, got [%T]", payloadBoxed)
	}
	var tokens [][]byte
	if err := json.Unmarshal(raw, &tokens); err != nil {
		return nil, errors.Wrapf(err, "failed marshalling response")
	}

	return tokens, nil
}

func (n *Network) LocalMembership() *LocalMembership {
	return &LocalMembership{lm: n.n.LocalMembership()}
}

func GetInstance(sp view2.ServiceProvider, network, channel string) *Network {
	n := fabric.GetFabricNetworkService(sp, network)
	if n == nil {
		return nil
	}
	ch, err := n.Channel(channel)
	if err != nil {
		panic(fmt.Sprintf("cannot find channel [%s] for network [%s]", channel, network))
	}

	return &Network{
		n:  n,
		ch: ch,
		sp: sp,
	}
}
