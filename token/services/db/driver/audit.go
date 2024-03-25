/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
)

// AuditTransactionDB defines the interface for a database to store the audit records of token transactions.
type AuditTransactionDB interface {
	// Close closes the database
	Close() error

	// BeginUpdate begins a new update to the database
	BeginUpdate() error

	// Commit commits the current update to the database
	Commit() error

	// Discard discards the current update to the database
	Discard() error

	// SetStatus sets the status of a transaction
	SetStatus(txID string, status TxStatus, message string) error

	// GetStatus returns the status of a given transaction.
	// It returns an error if the transaction is not found
	GetStatus(txID string) (TxStatus, string, error)

	// AddMovement adds a movement record to the database.
	// Each token transaction can be seen as a list of movements.
	AddMovement(record *MovementRecord) error

	// AddTransaction adds a transaction record to the database.
	AddTransaction(record *TransactionRecord) error

	// QueryTransactions returns a list of transactions that match the given criteria
	QueryTransactions(params QueryTransactionsParams) (TransactionIterator, error)

	// QueryMovements returns a list of movement records
	QueryMovements(params QueryMovementsParams) ([]*MovementRecord, error)

	// QueryValidations returns a list of validation  records
	QueryValidations(params QueryValidationRecordsParams) (ValidationRecordsIterator, error)

	// AddTokenRequest binds the passed transaction id to the passed token request
	AddTokenRequest(txID string, tr []byte) error

	// GetTokenRequest returns the token request bound to the passed transaction id, if available.
	// It returns nil without error if the key is not found.
	GetTokenRequest(txID string) ([]byte, error)
}

// AuditDBDriver is the interface for an audit database driver
type AuditDBDriver interface {
	// Open opens an audit database connection
	Open(sp view2.ServiceProvider, tmsID token2.TMSID) (AuditTransactionDB, error)
}
