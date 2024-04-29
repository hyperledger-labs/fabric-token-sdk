/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabric

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/keys"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/services/chaincode"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/events"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	api2 "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

const (
	InvokeFunction            = "invoke"
	QueryPublicParamsFunction = "queryPublicParams"
	QueryTokensFunctions      = "queryTokens"
	AreTokensSpent            = "areTokensSpent"
)

type NewVaultFunc = func(network, channel, namespace string) (vault.Vault, error)

type lm struct {
	lm *fabric.LocalMembership
}

func (n *lm) DefaultIdentity() view.Identity {
	return n.lm.DefaultIdentity()
}

func (n *lm) AnonymousIdentity() view.Identity {
	return n.lm.AnonymousIdentity()
}

type nv struct {
	v          *fabric.Vault
	tokenVault driver.TokenVault
}

func (v *nv) QueryEngine() api2.QueryEngine {
	return v.tokenVault.QueryEngine()
}

func (v *nv) CertificationStorage() vault.CertificationStorage {
	return v.tokenVault.CertificationStorage()
}

func (v *nv) DeleteTokens(ids ...*token.ID) error {
	return v.tokenVault.DeleteTokens(ids...)
}

func (v *nv) Status(id string) (driver.ValidationCode, string, error) {
	vc, message, err := v.v.Status(id)
	return vc, message, err
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

func (v *nv) DiscardTx(id string, message string) error {
	return v.v.DiscardTx(id, message)
}

type ledger struct {
	l *fabric.Ledger
}

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

type Network struct {
	n           *fabric.NetworkService
	ch          *fabric.Channel
	tmsProvider *token2.ManagementServiceProvider
	viewManager *view2.Manager
	ledger      *ledger

	vaultCacheLock sync.RWMutex
	vaultCache     map[string]driver.Vault
	NewVault       NewVaultFunc
	subscribers    *events.Subscribers
}

func NewNetwork(sp view2.ServiceProvider, n *fabric.NetworkService, ch *fabric.Channel, newVault NewVaultFunc) *Network {
	return &Network{
		n:           n,
		ch:          ch,
		tmsProvider: token2.GetManagementServiceProvider(sp),
		viewManager: view2.GetManager(sp),
		ledger:      &ledger{ch.Ledger()},
		vaultCache:  map[string]driver.Vault{},
		NewVault:    newVault,
		subscribers: events.NewSubscribers(),
	}
}

func (n *Network) Name() string {
	return n.n.Name()
}

func (n *Network) Channel() string {
	return n.ch.Name()
}

func (n *Network) Vault(namespace string) (driver.Vault, error) {
	if len(namespace) == 0 {
		tms, err := n.tmsProvider.GetManagementService(token2.WithNetwork(n.n.Name()), token2.WithChannel(n.ch.Name()))
		if tms == nil || err != nil {
			return nil, errors.Errorf("empty namespace passed, cannot find TMS for [%s:%s]: %v", n.n.Name(), n.ch.Name(), err)
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

	tokenVault, err := n.NewVault(n.Name(), n.Channel(), namespace)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get token vault")
	}
	nv := &nv{
		v:          n.ch.Vault(),
		tokenVault: tokenVault,
	}
	// store in cache
	n.vaultCache[namespace] = nv

	return nv, nil
}

func (n *Network) StoreEnvelope(env driver.Envelope) error {
	rws, err := n.ch.Vault().GetRWSet(env.TxID(), env.Results())
	if err != nil {
		return errors.WithMessagef(err, "failed to get rwset")
	}
	rws.Done()

	rawEnv, err := env.Bytes()
	if err != nil {
		return errors.WithMessagef(err, "failed marshalling tx env [%s]", env.TxID())
	}

	return n.ch.Vault().StoreEnvelope(env.TxID(), rawEnv)
}

func (n *Network) EnvelopeExists(id string) bool {
	return n.ch.EnvelopeService().Exists(id)
}

func (n *Network) Broadcast(context context.Context, blob interface{}) error {
	return n.n.Ordering().Broadcast(context, blob)
}

func (n *Network) NewEnvelope() driver.Envelope {
	return n.n.TransactionManager().NewEnvelope()
}

func (n *Network) StoreTransient(id string, transient driver.TransientMap) error {
	return n.ch.Vault().StoreTransient(id, fabric.TransientMap(transient))
}

func (n *Network) TransientExists(id string) bool {
	return n.ch.MetadataService().Exists(id)
}

func (n *Network) GetTransient(id string) (driver.TransientMap, error) {
	tm, err := n.ch.MetadataService().LoadTransient(id)
	if err != nil {
		return nil, err
	}
	return driver.TransientMap(tm), nil
}

func (n *Network) RequestApproval(context view.Context, tms *token2.ManagementService, requestRaw []byte, signer view.Identity, txID driver.TxID) (driver.Envelope, error) {
	env, err := chaincode.NewEndorseView(
		tms.Namespace(),
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
	ppBoxed, err := n.viewManager.InitiateView(
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

func (n *Network) QueryTokens(context view.Context, namespace string, IDs []*token.ID) ([][]byte, error) {
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
		return nil, errors.WithMessagef(err, "failed to query the token chaincode for tokens")
	}

	// Unbox
	raw, ok := payloadBoxed.([]byte)
	if !ok {
		return nil, errors.Errorf("expected []byte from TCC, got [%T]", payloadBoxed)
	}
	var tokens [][]byte
	if err := json.Unmarshal(raw, &tokens); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal response")
	}

	return tokens, nil
}

func (n *Network) AreTokensSpent(c view.Context, namespace string, tokenIDs []*token.ID, meta []string) ([]bool, error) {
	sIDs := make([]string, len(tokenIDs))
	var err error
	for i, id := range tokenIDs {
		sIDs[i], err = keys.CreateTokenKey(id.TxId, id.Index)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot compute spent id for [%v]", id)
		}
	}

	idsRaw, err := json.Marshal(sIDs)
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshalling ids")
	}

	payloadBoxed, err := c.RunView(chaincode.NewQueryView(
		namespace,
		AreTokensSpent,
		idsRaw,
	).WithNetwork(n.Name()).WithChannel(n.Channel()))
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to query the token chaincode for tokens spent")
	}

	// Unbox
	raw, ok := payloadBoxed.([]byte)
	if !ok {
		return nil, errors.Errorf("expected []byte from TCC, got [%T]", payloadBoxed)
	}
	var spent []bool
	if err := json.Unmarshal(raw, &spent); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal esponse")
	}

	return spent, nil
}

func (n *Network) LocalMembership() driver.LocalMembership {
	return &lm{
		lm: n.n.LocalMembership(),
	}
}

func (n *Network) AddFinalityListener(namespace string, txID string, listener driver.FinalityListener) error {
	wrapper := &FinalityListener{
		net:       n,
		root:      listener,
		network:   n.n.Name(),
		namespace: namespace,
	}
	n.subscribers.Set(txID, listener, wrapper)
	return n.ch.Committer().AddFinalityListener(txID, wrapper)
}

func (n *Network) RemoveFinalityListener(txID string, listener driver.FinalityListener) error {
	wrapper, ok := n.subscribers.Get(txID, listener)
	if !ok {
		return errors.Errorf("listener was not registered")
	}
	return n.ch.Committer().RemoveFinalityListener(txID, wrapper.(*FinalityListener))
}

func (n *Network) LookupTransferMetadataKey(namespace string, startingTxID string, key string, timeout time.Duration) ([]byte, error) {
	transferMetadataKey, err := keys.CreateTransferActionMetadataKey(key)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to generate transfer action metadata key from [%s]", key)
	}
	var keyValue []byte
	c, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	v := n.ch.Vault()
	if err := n.ch.Delivery().Scan(c, startingTxID, func(tx *fabric.ProcessedTransaction) (bool, error) {
		logger.Debugf("scanning [%s]...", tx.TxID())

		rws, err := v.GetEphemeralRWSet(tx.Results())
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

func (n *Network) ProcessNamespace(namespace string) error {
	if err := n.ch.Committer().ProcessNamespace(namespace); err != nil {
		return errors.WithMessagef(err, "failed to register processing of namespace [%s]", namespace)
	}
	return nil
}

type FinalityListener struct {
	net       *Network
	root      driver.FinalityListener
	network   string
	namespace string
}

func (t *FinalityListener) OnStatus(txID string, status int, message string) {
	defer func() {
		if e := recover(); e != nil {
			logger.Debugf("failed finality update for tx [%s]: [%s]", txID, e)
			if err := t.net.AddFinalityListener(txID, t.namespace, t.root); err != nil {
				panic(err)
			}
			logger.Debugf("added finality listener for tx [%s]...done", txID)
		}
	}()

	key, err := keys.CreateTokenRequestKey(txID)
	if err != nil {
		panic(fmt.Sprintf("can't create for token request [%s]", txID))
	}

	v := t.net.ch.Vault()
	qe, err := v.NewQueryExecutor()
	if err != nil {
		panic(fmt.Sprintf("can't get query executor [%s]", txID))
	}
	tokenRequestHash, err := qe.GetState(t.namespace, key)
	if err != nil {
		panic(fmt.Sprintf("can't get state [%s][%s]", txID, key))
	}
	if err != nil {
		qe.Done()
	}
	qe.Done()
	t.root.OnStatus(txID, status, message, tokenRequestHash)
}
