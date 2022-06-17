/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"math/big"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
)

// TransactionType is the type of transaction
type TransactionType int

const (
	// Issue is the type of transaction for issuing assets
	Issue TransactionType = iota
	// Transfer is the type of transaction for transferring assets
	Transfer
	// Redeem is the type of transaction for redeeming assets
	Redeem
)

// SearchDirection defines the direction of a search.
type SearchDirection int

const (
	// FromLast defines the direction of a search from the last key.
	FromLast SearchDirection = iota
	// FromBeginning defines the direction of a search from the first key.
	FromBeginning
)

// MovementDirection defines the direction of a movement.
type MovementDirection int

const (
	// Sent amount transferred from.
	Sent MovementDirection = iota
	// Received amount transferred to.
	Received
	// All amount transferred to and from.
	All
)

// TxStatus is the status of a transaction
type TxStatus string

const (
	// Pending is the status of a transaction that has been submitted to the ledger
	Pending TxStatus = "Pending"
	// Confirmed is the status of a transaction that has been confirmed by the ledger
	Confirmed TxStatus = "Confirmed"
	// Deleted is the status of a transaction that has been deleted due to a failure to commit
	Deleted TxStatus = "Deleted"
)

// MovementRecord is a record of a movement
type MovementRecord struct {
	// TxID is the transaction ID
	TxID string
	// EnrollmentID is the enrollment ID of the account that is receiving or sendeing
	EnrollmentID string
	// TokenType is the type of token
	TokenType string
	// Amount is positive if tokens are received. Negative otherwise
	Amount *big.Int
	// Status is the status of the transaction
	Status TxStatus
}

// TransactionRecord is the record of a transaction
type TransactionRecord struct {
	// TxID is the transaction ID
	TxID string
	// TransactionType is the type of transaction
	TransactionType TransactionType
	// SenderEID is the enrollment ID of the account that is sending tokens
	SenderEID string
	// RecipientEID is the enrollment ID of the account that is receiving tokens
	RecipientEID string
	// TokenType is the type of token
	TokenType string
	// Amount is positive if tokens are received. Negative otherwise
	Amount *big.Int
	// Timestamp is the time the transaction was submitted to the db
	Timestamp time.Time
	// Status is the status of the transaction
	Status TxStatus
}

// TransactionIterator is an iterator for transactions
type TransactionIterator interface {
	Close()
	Next() (*TransactionRecord, error)
}

// QueryMovementsParams defines the parameters for querying movements
type QueryMovementsParams struct {
	// EnrollmentIDs is the enrollment IDs of the accounts to query
	EnrollmentIDs []string
	// TokenTypes is the token types to query
	TokenTypes []string
	// TxStatuses is the statuses of the transactions to query
	TxStatuses []TxStatus
	// SearchDirection is the direction of the search
	SearchDirection SearchDirection
	// MovementDirection is the direction of the movement
	MovementDirection MovementDirection
	// NumRecords is the number of records to return
	// If 0, all records are returned
	NumRecords int
}

// QueryTransactionsParams defines the parameters for querying transactions
type QueryTransactionsParams struct {
	// SenderWallet is the wallet of the sender
	// If empty, any sender is accepted
	// If the sender does not match but the recipient matches, the transaction is returned
	SenderWallet string
	// RecipientWallet is the wallet of the recipient
	// If empty, any recipient is accepted
	// If the recipient does not match but the sender matches, the transaction is returned
	RecipientWallet string
	// From is the start time of the query
	// If nil, the query starts from the first transaction
	From *time.Time
	// To is the end time of the query
	// If nil, the query ends at the last transaction
	To *time.Time
}

// TokenTransactionDB defines the interface for a token transactions related database
type TokenTransactionDB interface {
	// Close closes the database
	Close() error

	// BeginUpdate begins a new update to the database
	BeginUpdate() error

	// Commit commits the current update to the database
	Commit() error

	// Discard discards the current update to the database
	Discard() error

	// SetStatus sets the status of a transaction
	SetStatus(txID string, status TxStatus) error

	// AddMovement adds a movement record to the database
	AddMovement(record *MovementRecord) error

	// AddTransaction adds a transaction record to the database
	AddTransaction(record *TransactionRecord) error

	// QueryTransactions returns a list of transactions that match the given criteria
	QueryTransactions(params QueryTransactionsParams) (TransactionIterator, error)

	// QueryMovements returns a list of movement records
	QueryMovements(params QueryMovementsParams) ([]*MovementRecord, error)
}

// Driver is the interface for a database driver
type Driver interface {
	// Open opens a database connection
	Open(sp view.ServiceProvider, name string) (TokenTransactionDB, error)
}
