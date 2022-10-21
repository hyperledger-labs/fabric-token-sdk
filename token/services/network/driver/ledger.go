/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

// Ledger models the ledger service
type Ledger interface {
	// Status returns the status of the transaction
	Status(id string) (ValidationCode, error)
}
