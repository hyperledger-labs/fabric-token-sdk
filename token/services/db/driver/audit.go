/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token"
)

// AuditTransactionDB defines the interface for a database to store the audit records of token transactions.
type AuditTransactionDB interface {
	// Close closes the database
	Close() error

	// BeginAtomicWrite opens an atomic database transaction. It must be committed or discarded.
	BeginAtomicWrite() (AtomicWrite, error)

	// SetStatus sets the status of a TokenRequest
	// (and with that, the associated ValidationRecord, Movement and Transaction)
	SetStatus(txID string, status TxStatus, message string) error

	// GetStatus returns the status of a given transaction.
	// It returns an error if the transaction is not found
	GetStatus(txID string) (TxStatus, string, error)

	// QueryTransactions returns a list of transactions that match the passed params
	QueryTransactions(params QueryTransactionsParams) (TransactionIterator, error)

	// QueryMovements returns a list of movement records
	QueryMovements(params QueryMovementsParams) ([]*MovementRecord, error)

	// QueryValidations returns an iterator over the validation records matching the passed params
	QueryValidations(params QueryValidationRecordsParams) (ValidationRecordsIterator, error)

	// QueryTokenRequests returns an iterator over the token requests matching the passed params
	QueryTokenRequests(params QueryTokenRequestsParams) (TokenRequestIterator, error)

	// GetTokenRequest returns the token request bound to the passed transaction id, if available.
	// It returns nil without error if the key is not found.
	GetTokenRequest(txID string) ([]byte, error)
}

// AuditDBDriver is the interface for an audit database driver
type AuditDBDriver interface {
	// Open opens an audit database connection
	Open(cp ConfigProvider, tmsID token.TMSID) (AuditTransactionDB, error)
}
