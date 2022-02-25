/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabric

import (
	"encoding/json"
	"strings"
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	idemix2 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/msp/idemix"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/services/chaincode"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

const (
	IdemixMSP = "idemix"
	BccspMSP  = "bccsp"

	InvokeFunction            = "invoke"
	QueryPublicParamsFunction = "queryPublicParams"
	QueryTokensFunctions      = "queryTokens"
)

type GetFunc func() (view.Identity, []byte, error)

type lm struct {
	lm *fabric.LocalMembership
}

func (n *lm) DefaultIdentity() view.Identity {
	return n.lm.DefaultIdentity()
}

func (n *lm) AnonymousIdentity() view.Identity {
	return n.lm.AnonymousIdentity()
}

func (n *lm) IsMe(id view.Identity) bool {
	return n.lm.IsMe(id)
}

func (n *lm) GetAnonymousIdentity(label string, auditInfo []byte) (string, string, driver.GetFunc, error) {
	if idInfo := n.lm.GetIdentityInfoByLabel(IdemixMSP, label); idInfo != nil {
		ai := auditInfo
		return idInfo.ID, idInfo.EnrollmentID, func() (view.Identity, []byte, error) {
			opts := []fabric.IdentityOption{fabric.WithIdemixEIDExtension()}
			if len(auditInfo) != 0 {
				opts = append(opts, fabric.WithAuditInfo(ai))
			}
			return idInfo.GetIdentity(opts...)
		}, nil
	}
	return "", "", nil, errors.New("not found")
}

func (n *lm) GetAnonymousIdentifier(label string) (string, error) {
	if idInfo := n.lm.GetIdentityInfoByLabel(IdemixMSP, label); idInfo != nil {
		return idInfo.ID, nil
	}
	return "", errors.New("not found")
}

func (n *lm) GetLongTermIdentity(label string) (string, string, view.Identity, error) {
	if idInfo := n.lm.GetIdentityInfoByLabel(BccspMSP, label); idInfo != nil {
		id, _, err := idInfo.GetIdentity()
		if err != nil {
			return "", "", nil, errors.New("failed to get identity")
		}
		return idInfo.ID, idInfo.EnrollmentID, id, err
	}
	return "", "", nil, errors.New("not found")
}

func (n *lm) GetLongTermIdentifier(id view.Identity) (string, error) {
	if idInfo := n.lm.GetIdentityInfoByIdentity(BccspMSP, id); idInfo != nil {
		return idInfo.ID, nil
	}
	return "", errors.New("not found")
}

func (n *lm) RegisterIdentity(id string, typ string, path string) error {
	// split type in type and msp id
	typeAndMspID := strings.Split(typ, ":")
	if len(typeAndMspID) < 2 {
		return errors.Errorf("invalid identity type '%s'", typ)
	}

	switch typeAndMspID[0] {
	case IdemixMSP:
		return n.lm.RegisterIdemixMSP(id, path, typeAndMspID[1])
	case BccspMSP:
		return n.lm.RegisterX509MSP(id, path, typeAndMspID[1])
	default:
		return errors.Errorf("invalid identity type '%s'", typ)
	}
}

type nv struct {
	v          *fabric.Vault
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

type Network struct {
	n  *fabric.NetworkService
	ch *fabric.Channel
	sp view2.ServiceProvider

	vaultCacheLock sync.RWMutex
	vaultCache     map[string]driver.Vault
}

func NewNetwork(sp view2.ServiceProvider, n *fabric.NetworkService, ch *fabric.Channel) *Network {
	return &Network{n: n, ch: ch, sp: sp, vaultCache: map[string]driver.Vault{}}
}

func (n *Network) Name() string {
	return n.n.Name()
}

func (n *Network) Channel() string {
	return n.ch.Name()
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

	tokenVault := vault.New(n.sp, n.Channel(), namespace, NewVault(n.ch))
	nv := &nv{
		v:          n.ch.Vault(),
		tokenVault: tokenVault,
	}
	// store in cache
	n.vaultCache[namespace] = nv

	return nv, nil
}

func (n *Network) GetRWSet(id string, results []byte) (driver.RWSet, error) {
	rws, err := n.ch.Vault().GetRWSet(id, results)
	if err != nil {
		return nil, err
	}
	return rws, nil
}

func (n *Network) StoreEnvelope(id string, env []byte) error {
	return n.ch.Vault().StoreEnvelope(id, env)
}

func (n *Network) Broadcast(blob interface{}) error {
	return n.n.Ordering().Broadcast(blob)
}

func (n *Network) IsFinalForParties(id string, endpoints ...view.Identity) error {
	return n.ch.Finality().IsFinalForParties(id, endpoints...)
}

func (n *Network) IsFinal(id string) error {
	return n.ch.Finality().IsFinal(id)
}

func (n *Network) NewEnvelope() driver.Envelope {
	return n.n.TransactionManager().NewEnvelope()
}

func (n *Network) StoreTransient(id string, transient driver.TransientMap) error {
	return n.ch.Vault().StoreTransient(id, fabric.TransientMap(transient))
}

func (n *Network) RequestApproval(context view.Context, namespace string, requestRaw []byte, signer view.Identity, txID driver.TxID) (driver.Envelope, error) {
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

func (n *Network) LocalMembership() driver.LocalMembership {
	return &lm{
		lm: n.n.LocalMembership(),
	}
}

func (n *Network) GetEnrollmentID(raw []byte) (string, error) {
	ai := &idemix2.AuditInfo{}
	if err := ai.FromBytes(raw); err != nil {
		return "", errors.Wrapf(err, "failed unamrshalling audit info [%s]", raw)
	}
	return ai.EnrollmentID(), nil
}
