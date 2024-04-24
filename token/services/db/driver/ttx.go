/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"errors"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
)

// TokenTransactionDB defines the interface for a token transaction database.
// This database is used to store records related to the processed token transactions.
type TokenTransactionDB interface {
	TransactionDB
	TransactionEndorsementAckDB
}

type AtomicWrite interface {
	// Commit commits the current update to the database
	Commit() error

	// Rollback discards the in progress database transaction.
	// It logs but otherwise ignores errors rolling back:
	// the result is always the end of the transaction.
	Rollback()

	// AddTokenRequest binds the passed transaction id to the passed token request
	AddTokenRequest(txID string, tr []byte) error

	// AddMovement adds a movement record to the database transaction.
	// Each token transaction can be seen as a list of movements.
	// This operation _requires_ a TokenRequest with the same tx_id to exist
	AddMovement(record *MovementRecord) error

	// AddTransaction adds a transaction record to the database transaction.
	// This operation _requires_ a TokenRequest with the same tx_id to exist
	AddTransaction(record *TransactionRecord) error

	// AddValidationRecord adds a new validation records for the given params
	// This operation _requires_ a TokenRequest with the same tx_id to exist
	AddValidationRecord(txID string, meta map[string][]byte) error
}

type TransactionDB interface {
	// Close closes the databases
	Close() error

	// BeginAtomicWrite opens an atomic database transaction. It must be committed or discarded.
	BeginAtomicWrite() (AtomicWrite, error)

	// SetStatus sets the status of a TokenRequest
	// (and with that, the associated ValidationRecord, Movement and Transaction)
	SetStatus(txID string, status TxStatus, message string) error

	// GetStatus returns the status of a given transaction.
	// It returns an error if the transaction is not found
	GetStatus(txID string) (TxStatus, string, error)

	// QueryTransactions returns a list of transactions that match the given criteria
	QueryTransactions(params QueryTransactionsParams) (TransactionIterator, error)

	// QueryMovements returns a list of movement records
	QueryMovements(params QueryMovementsParams) ([]*MovementRecord, error)

	// QueryValidations returns a list of validation  records
	QueryValidations(params QueryValidationRecordsParams) (ValidationRecordsIterator, error)

	// GetTokenRequest returns the token request bound to the passed transaction id, if available.
	// It returns nil without error if the key is not found.
	GetTokenRequest(txID string) ([]byte, error)
}

type TransactionEndorsementAckDB interface {
	// AddTransactionEndorsementAck records the signature of a given endorser for a given transaction
	AddTransactionEndorsementAck(txID string, endorser view.Identity, sigma []byte) error

	// GetTransactionEndorsementAcks returns the endorsement signatures for the given transaction id
	GetTransactionEndorsementAcks(txID string) (map[string][]byte, error)
}

// TTXDBDriver is the interface for a token transaction db driver
type TTXDBDriver interface {
	// Open opens a token transaction database
	Open(sp view2.ServiceProvider, tmsID token2.TMSID) (TokenTransactionDB, error)
}

var (
	ErrTokenRequestDoesNotExist = errors.New("token request does not exist")
)
