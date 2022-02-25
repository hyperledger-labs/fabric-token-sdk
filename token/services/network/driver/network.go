/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"encoding/base64"
	"fmt"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

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

	// Broadcast sends the passed blob to the network
	Broadcast(blob interface{}) error

	// IsFinalForParties takes in input a transaction id and an array of identities.
	// The identities are contacted to gather information about the finality of the
	// passed transaction
	IsFinalForParties(id string, endpoints ...view.Identity) error

	// IsFinal takes in input a transaction id and waits for its confirmation.
	IsFinal(id string) error

	// NewEnvelope returns a new instance of an envelope
	NewEnvelope() Envelope

	// StoreTransient stores the passed transient map and maps it to the passed id
	StoreTransient(id string, transient TransientMap) error

	// RequestApproval requests approval for the passed request and returns the returned envelope
	RequestApproval(context view.Context, namespace string, requestRaw []byte, signer view.Identity, txID TxID) (Envelope, error)

	// ComputeTxID computes the network transaction id from the passed abstract transaction id
	ComputeTxID(id *TxID) string

	// FetchPublicParameters returns the public parameters for the network.
	// If namespace is not supported, the argument is ignored.
	FetchPublicParameters(namespace string) ([]byte, error)

	// QueryTokens retrieves the token content for the passed token ids
	QueryTokens(context view.Context, namespace string, IDs []*token.ID) ([][]byte, error)

	// LocalMembership returns the local membership
	LocalMembership() LocalMembership

	// GetEnrollmentID returns the enrollment id of the passed identity
	GetEnrollmentID(raw []byte) (string, error)
}
