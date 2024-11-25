/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

// FinalityListener is the interface that must be implemented to receive transaction status change notifications
type FinalityListener interface {
	// OnStatus is called when the status of a transaction changes
	OnStatus(ctx context.Context, txID string, status int, message string, tokenRequestHash []byte)
}

type TransientMap = map[string][]byte

type TxID struct {
	Nonce   []byte
	Creator []byte
}

func (t *TxID) String() string {
	return fmt.Sprintf("[%s:%s]", base64.StdEncoding.EncodeToString(t.Nonce), base64.StdEncoding.EncodeToString(t.Creator))
}

// Network models a backend that stores tokens
type Network interface {
	// Name returns the name of the network
	Name() string

	// Channel returns the channel name, empty if not applicable
	Channel() string

	Normalize(opt *token2.ServiceOptions) (*token2.ServiceOptions, error)

	Connect(ns string) ([]token2.ServiceOption, error)

	// Vault returns the vault for the passed namespace. If namespaces are not supported,
	// the argument is ignored.
	Vault(namespace string) (Vault, error)

	TokenVault(namespace string) (TokenVault, error)

	// Broadcast sends the passed blob to the network
	Broadcast(context context.Context, blob interface{}) error

	// NewEnvelope returns a new instance of an envelope
	NewEnvelope() Envelope

	// RequestApproval requests approval for the passed request and returns the returned envelope
	RequestApproval(context view.Context, tms *token2.ManagementService, requestRaw []byte, signer view.Identity, txID TxID) (Envelope, error)

	// ComputeTxID computes the network transaction id from the passed abstract transaction id
	ComputeTxID(id *TxID) string

	// FetchPublicParameters returns the public parameters for the network.
	// If namespace is not supported, the argument is ignored.
	FetchPublicParameters(namespace string) ([]byte, error)

	// QueryTokens retrieves the token content for the passed token ids
	QueryTokens(context view.Context, namespace string, IDs []*token.ID) ([][]byte, error)

	// AreTokensSpent retrieves the spent flag for the passed ids
	AreTokensSpent(context view.Context, namespace string, tokenIDs []*token.ID, meta []string) ([]bool, error)

	// LocalMembership returns the local membership
	LocalMembership() LocalMembership

	// AddFinalityListener registers a listener for transaction status for the passed transaction id.
	// If the status is already valid or invalid, the listener is called immediately.
	// When the listener is invoked, then it is also removed.
	// If the transaction id is empty, the listener will be called on status changes of any transaction.
	// In this case, the listener is not removed
	AddFinalityListener(namespace string, txID string, listener FinalityListener) error

	// RemoveFinalityListener unregisters the passed listener.
	RemoveFinalityListener(id string, listener FinalityListener) error

	// LookupTransferMetadataKey searches for a transfer metadata key containing the passed sub-key starting from the passed transaction id in the given namespace.
	// The operation gets canceled if the passed timeout elapses or, if stopOnLastTx is true, when the last transaction in the vault is reached.
	LookupTransferMetadataKey(namespace string, startingTxID string, subKey string, timeout time.Duration, stopOnLastTx bool) ([]byte, error)

	// Ledger gives access to the remote ledger
	Ledger() (Ledger, error)

	// ProcessNamespace indicates to the commit pipeline to process all transaction in the passed namespace
	ProcessNamespace(namespace string) error
}

type Interoperability interface {
	InteropURL(namespace string) string
}
