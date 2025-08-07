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
	// GetStates returns the value corresponding to the given keys stored in the given namespace.
	GetStates(ctx context.Context, namespace string, keys ...string) ([][]byte, error)
}
