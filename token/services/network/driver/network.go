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

// TxStatusChangeListener is the interface that must be implemented to receive transaction status change notifications
type TxStatusChangeListener interface {
	// OnStatusChange is called when the status of a transaction changes
	OnStatusChange(txID string, status int) error
}

type TransientMap map[string][]byte

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

	// Vault returns the vault for the passed namespace. If namespaces are not supported,
	// the argument is ignored.
	Vault(namespace string) (Vault, error)

	// GetRWSet returns the read-write set for the passed id and marshalled set
	GetRWSet(id string, results []byte) (RWSet, error)

	// StoreEnvelope stores locally the passed envelope mapping it to the passed id
	StoreEnvelope(id string, env []byte) error

	// EnvelopeExists returns true if an envelope exists for the passed id, false otherwise
	EnvelopeExists(id string) bool

	// Broadcast sends the passed blob to the network
	Broadcast(context context.Context, blob interface{}) error

	// IsFinalForParties takes in input a transaction id and an array of identities.
	// The identities are contacted to gather information about the finality of the
	// passed transaction
	IsFinalForParties(id string, endpoints ...view.Identity) error

	// IsFinal takes in input a transaction id and waits for its confirmation
	// with the respect to the passed context that can be used to set a deadline
	// for the waiting time.
	IsFinal(ctx context.Context, id string) error

	// NewEnvelope returns a new instance of an envelope
	NewEnvelope() Envelope

	// StoreTransient stores the passed transient map and maps it to the passed id
	StoreTransient(id string, transient TransientMap) error

	// TransientExists returns true if a transient map exists for the passed id, false otherwise
	TransientExists(id string) bool

	// GetTransient retrieves the transient map bound to the passed id
	GetTransient(id string) (TransientMap, error)

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
	AreTokensSpent(context view.Context, namespace string, IDs []string) ([]bool, error)

	// LocalMembership returns the local membership
	LocalMembership() LocalMembership

	// GetEnrollmentID returns the enrollment id of the passed identity
	GetEnrollmentID(raw []byte) (string, error)

	// SubscribeTxStatusChanges registers a listener for transaction status changes for the passed id
	SubscribeTxStatusChanges(txID string, listener TxStatusChangeListener) error

	// UnsubscribeTxStatusChanges unregisters a listener for transaction status changes for the passed id
	UnsubscribeTxStatusChanges(id string, listener TxStatusChangeListener) error

	// LookupTransferMetadataKey searches for a transfer metadata key containing the passed sub-key starting from the passed transaction id in the given namespace.
	// The operation gets canceled if the passed timeout elapses.
	LookupTransferMetadataKey(namespace string, startingTxID string, subKey string, timeout time.Duration) ([]byte, error)

	// Ledger gives access to the remote ledger
	Ledger() (Ledger, error)

	// ProcessNamespace indicates to the commit pipeline to process all transaction in the passed namespace
	ProcessNamespace(namespace string) error
}
