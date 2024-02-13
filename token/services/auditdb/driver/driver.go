/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb/driver"
)

// ActionType is the type of transaction
type ActionType = driver.ActionType

const (
	// Issue is the action type for issuing tokens.
	Issue ActionType = iota
	// Transfer is the action type for transferring tokens.
	Transfer
	// Redeem is the action type for redeeming tokens.
	Redeem
)

// SearchDirection defines the direction of a search.
type SearchDirection = driver.SearchDirection

const (
	// FromLast defines the direction of a search from the last key.
	FromLast SearchDirection = iota
	// FromBeginning defines the direction of a search from the first key.
	FromBeginning
)

// MovementDirection defines the direction of a movement.
type MovementDirection = driver.MovementDirection

const (
	// Sent amount transferred from.
	Sent MovementDirection = iota
	// Received amount transferred to.
	Received
	// All amount transferred to and from.
	All
)

// TxStatus is the status of a transaction
type TxStatus = ttxdb.TxStatus

const (
	// Unknown is the status of a transaction that is unknown
	Unknown TxStatus = "Unknown"
	// Pending is the status of a transaction that has been submitted to the ledger
	Pending TxStatus = "Pending"
	// Confirmed is the status of a transaction that has been confirmed by the ledger
	Confirmed TxStatus = "Confirmed"
	// Deleted is the status of a transaction that has been deleted due to a failure to commit
	Deleted TxStatus = "Deleted"
)

// MovementRecord is a record of a movement of assets.
// Given a Token Transaction, a movement record is created for each enrollment ID that participated in the transaction
// and each token type that was transferred.
// The movement record contains the total amount of the token type that was transferred to/from the enrollment ID
// in a given token transaction.
type MovementRecord = driver.MovementRecord

// TransactionRecord is a more finer-grained version of a movement record.
// Given a Token Transaction, for each token action in the Token Request,
// a transaction record is created for each unique enrollment ID found in the outputs.
// The transaction record contains the total amount of the token type that was transferred to/from that enrollment ID
// in that action.
type TransactionRecord = driver.TransactionRecord

// ValidationRecord is a record that contains information about the validation of a given token request
type ValidationRecord = driver.ValidationRecord

// TransactionIterator is an iterator for transactions
type TransactionIterator = driver.TransactionIterator

// ValidationRecordsIterator is an iterator for transactions
type ValidationRecordsIterator = driver.ValidationRecordsIterator

// QueryMovementsParams defines the parameters for querying movements.
// Movement records will be filtered by EnrollmentID, TokenType, and Status.
// SearchDirection tells if the search should start from the oldest to the newest records or vice versa.
// MovementDirection which amounts to consider. Sent correspond to a negative amount,
// Received to a positive amount, and All to both.
type QueryMovementsParams = driver.QueryMovementsParams

// QueryTransactionsParams defines the parameters for querying transactions.
// One can filter by sender, by recipient, and by time range.
type QueryTransactionsParams = driver.QueryTransactionsParams

// QueryValidationRecordsParams defines the parameters for querying validation records.
type QueryValidationRecordsParams = driver.QueryValidationRecordsParams

// AuditTransactionDB defines the interface for a token transactions database
type AuditTransactionDB interface {
	driver.TransactionDB
}

// Driver is the interface for a database driver
type Driver interface {
	// Open opens a database connection
	Open(sp view2.ServiceProvider, tmsID token2.TMSID) (AuditTransactionDB, error)
}
