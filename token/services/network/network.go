/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package network

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type ValidationCode int

const (
	_               ValidationCode = iota
	Valid                          // Transaction is valid and committed
	Invalid                        // Transaction is invalid and has been discarded
	Busy                           // Transaction does not yet have a validity state
	Unknown                        // Transaction is unknown
	HasDependencies                // Transaction is unknown but has known dependencies
)

type UnspentTokensIterator = driver.UnspentTokensIterator

// TxStatusChangeListener is the interface that must be implemented to receive transaction status change notifications
type TxStatusChangeListener interface {
	// OnStatusChange is called when the status of a transaction changes
	OnStatusChange(txID string, status int) error
}

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

func (e *Envelope) String() string {
	return e.e.String()
}

type RWSet struct {
	rws driver.RWSet
}

func (s *RWSet) Done() {
	s.rws.Done()
}

type Vault struct {
	n  *Network
	v  driver.Vault
	ns string
}

func (v *Vault) GetLastTxID() (string, error) {
	return v.v.GetLastTxID()
}

func (v *Vault) UnspentTokensIteratorBy(id, typ string) (UnspentTokensIterator, error) {
	return v.v.UnspentTokensIteratorBy(id, typ)
}

func (v *Vault) UnspentTokensIterator() (UnspentTokensIterator, error) {
	return v.v.UnspentTokensIterator()
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

func (v *Vault) Status(id string) (ValidationCode, error) {
	vc, err := v.v.Status(id)
	return ValidationCode(vc), err
}

func (v *Vault) DiscardTx(id string) error {
	return v.v.DiscardTx(id)
}

// PruneInvalidUnspentTokens checks that each unspent token is actually available on the ledger.
// Those that are not available are deleted.
// The function returns the list of deleted token ids
func (v *Vault) PruneInvalidUnspentTokens(context view.Context) ([]*token2.ID, error) {
	it, err := v.UnspentTokensIterator()
	if err != nil {
		return nil, errors.WithMessage(err, "failed to get an iterator of unspent tokens")
	}

	var deleted []*token2.ID
	tms := token.GetManagementService(context, token.WithTMS(v.n.Name(), v.n.Channel(), v.ns))
	var buffer []*token2.UnspentToken
	bufferSize := 50
	for {
		tok, err := it.Next()
		if err != nil {
			return nil, errors.WithMessagef(err, "failed to get next unspent token")
		}
		if tok == nil {
			break
		}
		buffer = append(buffer, tok)
		if len(buffer) > bufferSize {
			newDeleted, err := v.deleteTokens(context, tms, buffer)
			if err != nil {
				return nil, errors.WithMessagef(err, "failed to process tokens [%v]", buffer)
			}
			deleted = append(deleted, newDeleted...)
			buffer = nil
		}
	}
	newDeleted, err := v.deleteTokens(context, tms, buffer)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to process tokens [%v]", buffer)
	}
	deleted = append(deleted, newDeleted...)

	return deleted, nil
}

func (v *Vault) deleteTokens(context view.Context, tms *token.ManagementService, tokens []*token2.UnspentToken) ([]*token2.ID, error) {
	logger.Debugf("delete tokens from vault [%d][%v]", len(tokens), tokens)
	if len(tokens) == 0 {
		return nil, nil
	}

	// get spent flags
	ids := make([]*token2.ID, len(tokens))
	for i, tok := range tokens {
		ids[i] = tok.Id
	}
	spentIDs, err := tms.WalletManager().SpentIDs(ids)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to compute spent ids for [%v]", ids)
	}
	spent, err := v.n.AreTokensSpent(context, tms.Namespace(), spentIDs)
	if err != nil {
		return nil, errors.WithMessagef(err, "cannot fetch spent flags from network [%s:%s] for ids [%v]", tms.Network(), tms.Channel(), ids)
	}

	// remove the tokens flagged as spent
	var toDelete []*token2.ID
	for i, tok := range tokens {
		if spent[i] {
			logger.Debugf("token [%s] is spent", tok.Id)
			toDelete = append(toDelete, tok.Id)
		} else {
			logger.Debugf("token [%s] is not spent", tok.Id)
		}
	}
	if err := v.v.TokenVault().DeleteTokens(tms.Namespace(), toDelete...); err != nil {
		return nil, errors.WithMessagef(err, "failed to remove token ids [%v]", toDelete)
	}

	return toDelete, nil
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

type Ledger struct {
	l driver.Ledger
}

func (l *Ledger) Status(id string) (ValidationCode, error) {
	vc, err := l.l.Status(id)
	if err != nil {
		return 0, err
	}
	return ValidationCode(vc), nil
}

// Network provides access to the remote network
type Network struct {
	n driver.Network
}

// Name returns the name of the network
func (n *Network) Name() string {
	return n.n.Name()
}

// Channel returns the channel name
func (n *Network) Channel() string {
	return n.n.Channel()
}

// Vault returns the vault for the given namespace
func (n *Network) Vault(namespace string) (*Vault, error) {
	v, err := n.n.Vault(namespace)
	if err != nil {
		return nil, err
	}
	return &Vault{n: n, v: v, ns: namespace}, nil
}

// GetRWSet returns the read-write set unmarshalled from the given bytes and bound to the given id
func (n *Network) GetRWSet(id string, results []byte) (*RWSet, error) {
	rws, err := n.n.GetRWSet(id, results)
	if err != nil {
		return nil, err
	}
	return &RWSet{rws: rws}, nil
}

// StoreEnvelope stores locally the given transaction envelope and associated it with the given id
func (n *Network) StoreEnvelope(id string, env []byte) error {
	return n.n.StoreEnvelope(id, env)
}

func (n *Network) ExistEnvelope(id string) bool {
	return n.n.EnvelopeExists(id)
}

func (n *Network) ExistTransient(id string) bool {
	return n.n.TransientExists(id)
}

// Broadcast sends the given blob to the network
func (n *Network) Broadcast(context context.Context, blob interface{}) error {
	switch b := blob.(type) {
	case *Envelope:
		return n.n.Broadcast(context, b.e)
	default:
		return n.n.Broadcast(context, b)
	}
}

// IsFinalForParties returns true if the given transaction is final for the given parties
func (n *Network) IsFinalForParties(id string, endpoints ...view.Identity) error {
	return n.n.IsFinalForParties(id, endpoints...)
}

// IsFinal returns true if the given transaction is final
func (n *Network) IsFinal(ctx context.Context, id string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	return n.n.IsFinal(ctx, id)
}

// AnonymousIdentity returns a fresh anonymous identity
func (n *Network) AnonymousIdentity() view.Identity {
	return n.n.LocalMembership().AnonymousIdentity()
}

// NewEnvelope creates a new envelope
func (n *Network) NewEnvelope() *Envelope {
	return &Envelope{e: n.n.NewEnvelope()}
}

// StoreTransient stores locally the given transient data and associated it with the given id
func (n *Network) StoreTransient(id string, transient TransientMap) error {
	return n.n.StoreTransient(id, driver.TransientMap(transient))
}

// GetTransient retrieves the transient map bound to the passed id
func (n *Network) GetTransient(id string) (TransientMap, error) {
	tm, err := n.n.GetTransient(id)
	if err != nil {
		return nil, err
	}
	return TransientMap(tm), nil
}

// RequestApproval requests approval for the given token request
func (n *Network) RequestApproval(context view.Context, tms *token.ManagementService, requestRaw []byte, signer view.Identity, txID TxID) (*Envelope, error) {
	env, err := n.n.RequestApproval(context, tms, requestRaw, signer, driver.TxID{
		Nonce:   txID.Nonce,
		Creator: txID.Creator,
	})
	if err != nil {
		return nil, err
	}
	return &Envelope{e: env}, nil
}

// ComputeTxID computes the transaction ID in the target network format for the given tx id
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

// FetchPublicParameters returns the public parameters for the given namespace
func (n *Network) FetchPublicParameters(namespace string) ([]byte, error) {
	return n.n.FetchPublicParameters(namespace)
}

// QueryTokens returns the tokens corresponding to the given token ids int the given namespace
func (n *Network) QueryTokens(context view.Context, namespace string, IDs []*token2.ID) ([][]byte, error) {
	return n.n.QueryTokens(context, namespace, IDs)
}

// AreTokensSpent retrieves the spent flag for the passed ids
func (n *Network) AreTokensSpent(context view.Context, namespace string, IDs []string) ([]bool, error) {
	return n.n.AreTokensSpent(context, namespace, IDs)
}

// LocalMembership returns the local membership for this network
func (n *Network) LocalMembership() *LocalMembership {
	return &LocalMembership{lm: n.n.LocalMembership()}
}

// GetEnrollmentID returns the enrollment ID bound to the passed marshalled audit info
func (n *Network) GetEnrollmentID(raw []byte) (string, error) {
	return n.n.GetEnrollmentID(raw)
}

// SubscribeTxStatusChanges register a listener for transaction status updates for the given id.
func (n *Network) SubscribeTxStatusChanges(txID string, listener TxStatusChangeListener) error {
	return n.n.SubscribeTxStatusChanges(txID, listener)
}

// UnsubscribeTxStatusChanges unregisters a listener for transaction status changes for the passed id
func (n *Network) UnsubscribeTxStatusChanges(id string, listener TxStatusChangeListener) error {
	return n.n.UnsubscribeTxStatusChanges(id, listener)
}

// LookupTransferMetadataKey searches for a transfer metadata key containing the passed sub-key starting from the passed transaction id in the given namespace.
// The operation gets canceled if the passed timeout gets reached.
func (n *Network) LookupTransferMetadataKey(namespace, startingTxID, key string, timeout time.Duration, opts ...token.ServiceOption) ([]byte, error) {
	return n.n.LookupTransferMetadataKey(namespace, startingTxID, key, timeout)
}

func (n *Network) Ledger(namespace string) (*Ledger, error) {
	l, err := n.n.Ledger()
	if err != nil {
		return nil, err
	}
	return &Ledger{l: l}, nil
}

func (n *Network) ProcessNamespace(namespace string) error {
	return n.n.ProcessNamespace(namespace)
}

// Provider returns an instance of network provider
type Provider struct {
	sp view2.ServiceProvider

	lock     sync.Mutex
	networks map[string]*Network
}

// NewProvider returns a new instance of network provider
func NewProvider(sp view2.ServiceProvider) *Provider {
	ms := &Provider{
		sp:       sp,
		networks: map[string]*Network{},
	}
	return ms
}

// GetNetwork returns a network instance for the given network and channel
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
			logger.Warningf("failed to create network [%s:%s]: %s", network, channel, err)
			continue
		}
		if nw != nil {
			return &Network{n: nw}, nil
		}
	}
	return nil, errors.Errorf("no network driver found for [%s:%s]", network, channel)
}

// GetInstance returns a network instance for the given network and channel
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
