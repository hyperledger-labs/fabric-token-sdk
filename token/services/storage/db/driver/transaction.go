/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

// TransactionImpl is an alias for a generic transaction implementation type
type TransactionImpl = any

// Transaction defines an atomic transaction
type Transaction interface {
	Impl() TransactionImpl
	// Commit applies and persists any pending changes to the asset database.
	Commit() error
	// Rollback undoes all pending changes, restoring the asset database to its previous committed state.
	Rollback()
}
