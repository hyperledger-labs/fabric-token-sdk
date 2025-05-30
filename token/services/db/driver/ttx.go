/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"context"
	"errors"

	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

// TokenTransactionStore defines the interface for a token transaction database.
// This database is used to store records related to the processed token transactions.
type TokenTransactionStore interface {
	TransactionStore
	TransactionEndorsementAckStore
}

type AtomicWrite interface {
	// Commit commits the current update to the database
	Commit() error

	// Rollback discards the in progress database transaction.
	// It logs but otherwise ignores errors rolling back:
	// the result is always the end of the transaction.
	Rollback()

	// AddTokenRequest binds the passed transaction id to the passed token request
	AddTokenRequest(ctx context.Context, txID string, tr []byte, applicationMetadata, publicMetadata map[string][]byte, ppHash driver.PPHash) error

	// AddMovement adds a movement record to the database transaction.
	// Each token transaction can be seen as a list of movements.
	// This operation _requires_ a TokenRequest with the same tx_id to exist
	AddMovement(ctx context.Context, record *MovementRecord) error

	// AddTransaction adds a transaction record to the database transaction.
	// This operation _requires_ a TokenRequest with the same tx_id to exist
	AddTransaction(ctx context.Context, record *TransactionRecord) error

	// AddValidationRecord adds a new validation records for the given params
	// This operation _requires_ a TokenRequest with the same tx_id to exist
	AddValidationRecord(ctx context.Context, txID string, meta map[string][]byte) error
}

type TransactionStore interface {
	// Close closes the databases
	Close() error

	// BeginAtomicWrite opens an atomic database transaction. It must be committed or discarded.
	BeginAtomicWrite() (AtomicWrite, error)

	// SetStatus sets the status of a TokenRequest
	// (and with that, the associated ValidationRecord, Movement and Transaction)
	SetStatus(ctx context.Context, txID string, status TxStatus, message string) error

	// GetStatus returns the status of a given transaction.
	// It returns an error if the transaction is not found
	GetStatus(ctx context.Context, txID string) (TxStatus, string, error)

	// QueryTransactions returns a list of transactions that match the given criteria

	QueryTransactions(ctx context.Context, params QueryTransactionsParams, pagination driver2.Pagination) (*driver2.PageIterator[*TransactionRecord], error)

	// QueryMovements returns a list of movement records
	QueryMovements(ctx context.Context, params QueryMovementsParams) ([]*MovementRecord, error)

	// QueryValidations returns a list of validation  records
	QueryValidations(ctx context.Context, params QueryValidationRecordsParams) (ValidationRecordsIterator, error)

	// QueryTokenRequests returns an iterator over the token requests matching the passed params
	QueryTokenRequests(ctx context.Context, params QueryTokenRequestsParams) (TokenRequestIterator, error)

	// GetTokenRequest returns the token request bound to the passed transaction id, if available.
	// It returns nil without error if the key is not found.
	GetTokenRequest(ctx context.Context, txID string) ([]byte, error)
}

type TransactionEndorsementAckStore interface {
	// AddTransactionEndorsementAck records the signature of a given endorser for a given transaction
	AddTransactionEndorsementAck(ctx context.Context, txID string, endorser token.Identity, sigma []byte) error

	// GetTransactionEndorsementAcks returns the endorsement signatures for the given transaction id
	GetTransactionEndorsementAcks(ctx context.Context, txID string) (map[string][]byte, error)
}

var (
	ErrTokenRequestDoesNotExist = errors.New("token request does not exist")
)
