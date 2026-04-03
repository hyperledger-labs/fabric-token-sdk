/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"context"
)

// Ledger models the ledger service
type Ledger interface {
	// Status returns the status of the transaction
	Status(id string) (ValidationCode, error)
	// GetTransactionStatus retrieves the current status and token request hash for a transaction.
	// Returns the validation status, token request hash, status message, and any error encountered.
	GetTransactionStatus(ctx context.Context, namespace, txID string) (status int, tokenRequestHash []byte, message string, err error)
	// GetStates returns the value corresponding to the given keys stored in the given namespace.
	GetStates(ctx context.Context, namespace string, keys ...string) ([][]byte, error)
	// TransferMetadataKey returns the transfer metadata key associated to the given key
	TransferMetadataKey(k string) (string, error)
}
