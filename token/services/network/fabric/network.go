/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabric

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/hyperledger/fabric-protos-go/peer"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	idemix2 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/msp/idemix"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/services/chaincode"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/keys"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
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

func (v *nv) Status(txID string) (driver.ValidationCode, error) {
	vc, _, err := v.v.Status(txID)
	return driver.ValidationCode(vc), err
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

func (v *nv) DiscardTx(txID string) error {
	return v.v.DiscardTx(txID)
}

type ledger struct {
	l *fabric.Ledger
}

func (l *ledger) Status(id string) (driver.ValidationCode, error) {
	tx, err := l.l.GetTransactionByID(id)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to get transaction [%s]", id)
	}
	switch peer.TxValidationCode(tx.ValidationCode()) {
	case peer.TxValidationCode_VALID:
		return driver.Valid, nil
	default:
		return driver.Invalid, nil
	}
}

type Network struct {
	n      *fabric.NetworkService
	ch     *fabric.Channel
	sp     view2.ServiceProvider
	ledger *ledger

	vaultCacheLock sync.RWMutex
	vaultCache     map[string]driver.Vault
}

func NewNetwork(sp view2.ServiceProvider, n *fabric.NetworkService, ch *fabric.Channel) *Network {
	return &Network{
		n:          n,
		ch:         ch,
		sp:         sp,
		ledger:     &ledger{ch.Ledger()},
		vaultCache: map[string]driver.Vault{},
	}
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

func (n *Network) ExistEnvelope(id string) bool {
	return n.ch.EnvelopeService().Exists(id)
}

func (n *Network) Broadcast(blob interface{}) error {
	return n.n.Ordering().Broadcast(blob)
}

func (n *Network) IsFinalForParties(id string, endpoints ...view.Identity) error {
	return n.ch.Finality().IsFinalForParties(id, endpoints...)
}

func (n *Network) IsFinal(ctx context.Context, id string) error {
	return n.ch.Finality().IsFinal(ctx, id)
}

func (n *Network) NewEnvelope() driver.Envelope {
	return n.n.TransactionManager().NewEnvelope()
}

func (n *Network) StoreTransient(id string, transient driver.TransientMap) error {
	return n.ch.Vault().StoreTransient(id, fabric.TransientMap(transient))
}

func (n *Network) ExistTransient(id string) bool {
	return n.ch.MetadataService().Exists(id)
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

func (n *Network) SubscribeTxStatusChanges(txID string, listener driver.TxStatusChangeListener) error {
	return n.ch.Committer().SubscribeTxStatusChanges(txID, listener)
}

func (n *Network) UnsubscribeTxStatusChanges(txID string, listener driver.TxStatusChangeListener) error {
	return n.ch.Committer().UnsubscribeTxStatusChanges(txID, listener)
}

func (n *Network) LookupTransferMetadataKey(namespace string, startingTxID string, key string, timeout time.Duration) ([]byte, error) {
	transferMetadataKey, err := keys.CreateTransferActionMetadataKey(key)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to generate transfer action metadata key from [%s]", key)
	}
	var keyValue []byte
	c, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	vault := n.ch.Vault()
	if err := n.ch.Delivery().Scan(c, startingTxID, func(tx *fabric.ProcessedTransaction) (bool, error) {
		logger.Debugf("scanning [%s]...", tx.TxID())

		rws, err := vault.GetEphemeralRWSet(tx.Results())
		if err != nil {
			return false, err
		}

		found := false
		for _, ns := range rws.Namespaces() {
			if ns == namespace {
				found = true
				break
			}
		}
		if !found {
			logger.Debugf("scanning [%s] does not contain namespace [%s]", tx.TxID(), namespace)
			return false, nil
		}

		ns := namespace
		for i := 0; i < rws.NumWrites(ns); i++ {
			k, v, err := rws.GetWriteAt(ns, i)
			if err != nil {
				return false, err
			}
			if k == transferMetadataKey {
				keyValue = v
				return true, nil
			}
		}
		logger.Debugf("scanning for key [%s] on [%s] not found", transferMetadataKey, tx.TxID())
		return false, nil
	}); err != nil {
		if strings.Contains(err.Error(), "context done") {
			return nil, errors.WithMessage(err, "timeout reached")
		}
		return nil, err
	}

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("scanning for key [%s] with timeout [%s] found, [%s]",
			transferMetadataKey,
			timeout,
			base64.StdEncoding.EncodeToString(keyValue),
		)
	}
	return keyValue, nil
}

func (n *Network) Ledger() (driver.Ledger, error) {
	return n.ledger, nil
}
