/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ethereum

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type localMembership struct{}

func (l *localMembership) DefaultIdentity() view.Identity {
	return nil
}

func (l *localMembership) AnonymousIdentity() (view.Identity, error) {
	return nil, ErrNotImplemented
}

// Network is the first scaffold for an Ethereum/EVM-backed token network.
//
// The scaffold deliberately leaves ledger interaction, approval flow, finality, and query support
// for follow-up slices while already exposing a stable implementation of the driver.Network
// contract.
type Network struct {
	network         string
	channel         string
	localMembership driver.LocalMembership
}

// NewNetwork returns a new Ethereum network scaffold.
func NewNetwork(network, channel string) *Network {
	return &Network{
		network:         network,
		channel:         channel,
		localMembership: &localMembership{},
	}
}

// Name returns the configured network identifier.
func (n *Network) Name() string {
	return n.network
}

// Channel returns the configured channel or logical partition identifier.
func (n *Network) Channel() string {
	return n.channel
}

// Normalize fills missing network and channel values from the scaffold configuration.
func (n *Network) Normalize(opt *token2.ServiceOptions) (*token2.ServiceOptions, error) {
	if opt == nil {
		return nil, fmt.Errorf("service options are required")
	}
	if len(opt.Network) == 0 {
		opt.Network = n.network
	}
	if len(opt.Channel) == 0 {
		opt.Channel = n.channel
	}

	return opt, nil
}

// Connect is deferred to a later slice once a concrete Ethereum backend abstraction is introduced.
func (n *Network) Connect(ns string) ([]token2.ServiceOption, error) {
	return nil, ErrNotImplemented
}

// Broadcast is deferred to a later slice once a concrete Ethereum backend abstraction is introduced.
func (n *Network) Broadcast(ctx context.Context, blob interface{}) error {
	return ErrNotImplemented
}

// NewEnvelope returns an empty Ethereum envelope scaffold.
func (n *Network) NewEnvelope() driver.Envelope {
	return &Envelope{}
}

// RequestApproval is deferred to a later slice once the endorsement envelope and backend flow are defined.
func (n *Network) RequestApproval(context view.Context, tms *token2.ManagementService, requestRaw []byte, signer view.Identity, txID driver.TxID) (driver.Envelope, error) {
	return nil, ErrNotImplemented
}

// ComputeTxID returns a deterministic scaffold transaction identifier derived from the abstract TxID.
func (n *Network) ComputeTxID(id *driver.TxID) string {
	if id == nil {
		return ""
	}
	sum := sha256.Sum256(append(append([]byte{}, id.Nonce...), id.Creator...))

	return hex.EncodeToString(sum[:])
}

// FetchPublicParameters is deferred to a later slice once contract access is defined.
func (n *Network) FetchPublicParameters(namespace string) ([]byte, error) {
	return nil, ErrNotImplemented
}

// QueryTokens is deferred to a later slice once state query support is defined.
func (n *Network) QueryTokens(ctx context.Context, namespace string, IDs []*token.ID) ([][]byte, error) {
	return nil, ErrNotImplemented
}

// AreTokensSpent is deferred to a later slice once state query support is defined.
func (n *Network) AreTokensSpent(ctx context.Context, namespace string, tokenIDs []*token.ID, meta []string) ([]bool, error) {
	return nil, ErrNotImplemented
}

// LocalMembership returns the scaffold local membership implementation.
func (n *Network) LocalMembership() driver.LocalMembership {
	return n.localMembership
}

// AddFinalityListener is deferred to a later slice once finality integration is defined.
func (n *Network) AddFinalityListener(namespace string, txID string, listener driver.FinalityListener) error {
	return ErrNotImplemented
}

// GetTransactionStatus is deferred to a later slice once finality integration is defined.
func (n *Network) GetTransactionStatus(ctx context.Context, namespace, txID string) (status int, tokenRequestHash []byte, message string, err error) {
	return driver.Unknown, nil, "", ErrNotImplemented
}

// LookupTransferMetadataKey is deferred to a later slice once Ethereum metadata layout is defined.
func (n *Network) LookupTransferMetadataKey(namespace string, key string, timeout time.Duration) ([]byte, error) {
	return nil, ErrNotImplemented
}

// Ledger is deferred to a later slice once a ledger abstraction is introduced.
func (n *Network) Ledger() (driver.Ledger, error) {
	return nil, ErrNotImplemented
}
