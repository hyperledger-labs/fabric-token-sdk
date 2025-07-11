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
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/lazy"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type ValidationCode = driver.ValidationCode

const (
	Valid   = driver.Valid   // Transaction is valid and committed
	Invalid = driver.Invalid // Transaction is invalid and has been discarded
	Busy    = driver.Busy    // Transaction does not yet have a validity state
	Unknown = driver.Unknown // Transaction is unknown
)

var logger = logging.MustGetLogger()

// FinalityListener is the interface that must be implemented to receive transaction status change notifications
type FinalityListener interface {
	// OnStatus is called when the status of a transaction changes
	OnStatus(ctx context.Context, txID string, status int, message string, tokenRequestHash []byte)
}

type GetFunc func() (view.Identity, []byte, error)

type TxID struct {
	Nonce   []byte
	Creator []byte
}

func (t TxID) String() string {
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

func (e *Envelope) Bytes() ([]byte, error) {
	return e.e.Bytes()
}

func (e *Envelope) FromBytes(raw []byte) error {
	return e.e.FromBytes(raw)
}

func (e *Envelope) TxID() string {
	return e.e.TxID()
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

type LocalMembership struct {
	lm driver.LocalMembership
}

func (l *LocalMembership) DefaultIdentity() view.Identity {
	return l.lm.DefaultIdentity()
}

func (l *LocalMembership) AnonymousIdentity() (view.Identity, error) {
	return l.lm.AnonymousIdentity()
}

type Ledger struct {
	l driver.Ledger
}

func (l *Ledger) Status(id string) (ValidationCode, string, error) {
	vc, err := l.l.Status(id)
	if err != nil {
		return 0, "", err
	}
	return ValidationCode(vc), "", nil
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

// Broadcast sends the given blob to the network
func (n *Network) Broadcast(ctx context.Context, blob interface{}) error {
	switch b := blob.(type) {
	case *Envelope:
		return n.n.Broadcast(ctx, b.e)
	default:
		return n.n.Broadcast(ctx, b)
	}
}

// AnonymousIdentity returns a fresh anonymous identity
func (n *Network) AnonymousIdentity() (view.Identity, error) {
	return n.n.LocalMembership().AnonymousIdentity()
}

// NewEnvelope creates a new envelope
func (n *Network) NewEnvelope() *Envelope {
	return &Envelope{e: n.n.NewEnvelope()}
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
func (n *Network) QueryTokens(ctx context.Context, namespace string, IDs []*token2.ID) ([][]byte, error) {
	return n.n.QueryTokens(ctx, namespace, IDs)
}

// AreTokensSpent retrieves the spent flag for the passed ids
func (n *Network) AreTokensSpent(ctx context.Context, namespace string, tokenIDs []*token2.ID, meta []string) ([]bool, error) {
	return n.n.AreTokensSpent(ctx, namespace, tokenIDs, meta)
}

// LocalMembership returns the local membership for this network
func (n *Network) LocalMembership() *LocalMembership {
	return &LocalMembership{lm: n.n.LocalMembership()}
}

// AddFinalityListener registers a listener for transaction status for the passed transaction id.
// If the status is already valid or invalid, the listener is called immediately.
// When the listener is invoked, then it is also removed.
// If the transaction id is empty, the listener will be called on status changes of any transaction.
// In this case, the listener is not removed
func (n *Network) AddFinalityListener(namespace string, txID string, listener FinalityListener) error {
	return n.n.AddFinalityListener(namespace, txID, listener)
}

// RemoveFinalityListener unregisters the passed listener.
func (n *Network) RemoveFinalityListener(id string, listener FinalityListener) error {
	return n.n.RemoveFinalityListener(id, listener)
}

// LookupTransferMetadataKey searches for a transfer metadata key containing the passed sub-key starting from the passed transaction id in the given namespace.
// The operation gets canceled if the passed timeout gets reached or, if stopOnLastTx is true, when the last transaction in the vault is reached.
func (n *Network) LookupTransferMetadataKey(namespace, startingTxID, key string, timeout time.Duration, stopOnLastTx bool, opts ...token.ServiceOption) ([]byte, error) {
	return n.n.LookupTransferMetadataKey(namespace, startingTxID, key, timeout, stopOnLastTx)
}

func (n *Network) Ledger() (*Ledger, error) {
	l, err := n.n.Ledger()
	if err != nil {
		return nil, err
	}
	return &Ledger{l: l}, nil
}

func (n *Network) Normalize(opt *token.ServiceOptions) (*token.ServiceOptions, error) {
	return n.n.Normalize(opt)
}

func (n *Network) Connect(ns string) ([]token.ServiceOption, error) {
	return n.n.Connect(ns)
}

// Provider returns an instance of network provider
type Provider struct {
	networks        lazy.Provider[netId, *Network]
	networkProvider *networkProvider
}

type netId struct {
	network, channel string
}

func key(id netId) string {
	return id.network + id.channel
}

// NewProvider returns a new instance of network provider
func NewProvider() *Provider {
	ms := &networkProvider{drivers: make([]driver.Driver, 0)}

	return &Provider{
		networkProvider: ms,
		networks:        lazy.NewProviderWithKeyMapper(key, ms.newNetwork),
	}
}

func (np *Provider) RegisterDriver(driver driver.Driver) {
	np.networkProvider.registerDriver(driver)
}

// GetNetwork returns a network instance for the given network and channel
func (np *Provider) GetNetwork(network string, channel string) (*Network, error) {
	logger.Debugf("GetNetwork: [%s:%s]", network, channel)
	return np.networks.Get(netId{network: network, channel: channel})
}

func (np *Provider) Normalize(opt *token.ServiceOptions) (*token.ServiceOptions, error) {
	if opt == nil {
		return nil, errors.New("no service options provided")
	}
	n, err := np.GetNetwork(opt.Network, opt.Channel)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get network [%s:%s]", opt.Network, opt.Channel)
	}
	return n.Normalize(opt)
}

// networkProvider instantiates new networks based on the registered drivers
type networkProvider struct {
	drivers []driver.Driver
}

func (np *networkProvider) registerDriver(driver driver.Driver) {
	np.drivers = append(np.drivers, driver)
}

func (np *networkProvider) newNetwork(netId netId) (*Network, error) {
	network, channel := netId.network, netId.channel
	var errs []error
	for _, d := range np.drivers {
		logger.Debugf("new network service for [%s:%s]", network, channel)
		nw, err := d.New(network, channel)
		if err != nil {
			errs = append(errs, errors.WithMessagef(err, "failed to create network [%s:%s]", network, channel))
			continue
		}
		logger.Debugf("new network [%s:%s]", network, channel)
		return &Network{n: nw}, nil
	}
	return nil, errors.Errorf("no network driver found for [%s:%s], errs [%v]", network, channel, errs)
}

// GetInstance returns a network instance for the given network and channel
func GetInstance(sp token.ServiceProvider, network, channel string) *Network {
	n, err := GetProvider(sp).GetNetwork(network, channel)
	if err != nil {
		logger.Errorf("Failed to get network [%s:%s]: %s", network, channel, err)
		return nil
	}
	return n
}

func GetProvider(sp token.ServiceProvider) *Provider {
	s, err := sp.GetService(&Provider{})
	if err != nil {
		panic(fmt.Sprintf("Failed to get service: %s", err))
	}
	return s.(*Provider)
}
