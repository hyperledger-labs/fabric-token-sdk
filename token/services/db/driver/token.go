/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"context"
	"errors"
	"time"

	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/types/transaction"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type TokenRecord struct {
	// TxID is the ID of the transaction that created the token
	TxID string
	// Index is the index in the transaction
	Index uint64
	// IssuerRaw represents the serialization of the issuer identity
	// if this is an IssuedToken.
	IssuerRaw []byte
	// OwnerRaw is the serialization of the owner TypedIdentity
	OwnerRaw []byte
	// OwnerType is the deserialized type inside OwnerRaw
	OwnerType string
	// OwnerIdentity is the deserialized Identity inside OwnerRaw
	OwnerIdentity []byte
	// OwnerWalletID is the identifier of the wallet that owns this token, it might be empty
	OwnerWalletID string
	// Ledger is the raw token as stored on the ledger
	Ledger []byte
	// LedgerFormat is the type of the raw token as stored on the ledger
	LedgerFormat token.Format
	// LedgerMetadata is the metadata associated to the content of Ledger
	LedgerMetadata []byte
	// Quantity is the number of units of Type carried in the token.
	// It is encoded as a string containing a number in base 16. The string has prefix ``0x''.
	Quantity string
	// Type is the type of token
	Type token.Type
	// Amount is the Quantity converted to decimal
	Amount uint64
	// Owner is used to mark the token as owned by this node
	Owner bool
	// Auditor is used to mark this token as audited by this node
	Auditor bool
	// Issuer issued to mark this token as issued by this node
	Issuer bool
}

// TokenDetails provides details about an owned (spent or unspent) token
type TokenDetails struct {
	// TxID is the ID of the transaction that created the token
	TxID string
	// Index is the index in the transaction
	Index uint64
	// OwnerIdentity is the serialization of the owner identity
	OwnerIdentity []byte
	// OwnerType is the deserialized type inside OwnerRaw
	OwnerType string
	// OwnerEnrollment is the enrollment id of the owner
	OwnerEnrollment string
	// Type is the type of token
	Type string
	// Amount is the Quantity converted to decimal
	Amount uint64
	// IsSpent is true if the token has been spent
	IsSpent bool
	// SpentBy is the transactionID that spent this token, if available
	SpentBy string
	// StoredAt is the moment the token was stored by this wallet
	StoredAt time.Time
}

// QueryTokenDetailsParams defines the parameters for querying token details
type QueryTokenDetailsParams struct {
	// WalletID is the optional identifier of the wallet owning the token
	WalletID string
	// OwnerType is the type of owner, for instance 'idemix' or 'htlc'
	OwnerType string
	// TokenType (optional) is the type of token
	TokenType token.Type
	// IDs is an optional list of specific token ids to return
	IDs []*token.ID
	// TransactionIDs selects tokens that are the output of the provided transaction ids.
	TransactionIDs []string
	// IncludeDeleted determines whether to include spent tokens. It defaults to false.
	IncludeDeleted bool
	// Spendable determines whether to include only spendable/non-spendable or any tokens. It defaults to nil (any tokens)
	Spendable SpendableFilter
	// LedgerTokenFormats selects tokens whose output on the ledger has a format in the list
	LedgerTokenFormats []token.Format
}

type SpendableFilter int

const (
	Any SpendableFilter = iota
	SpendableOnly
	NonSpendableOnly
)

// CertificationDB defines a database to manager token certifications
type CertificationDB interface {
	// ExistsCertification returns true if a certification for the passed token exists,
	// false otherwise
	ExistsCertification(id *token.ID) bool

	// StoreCertifications stores the passed certifications
	StoreCertifications(certifications map[*token.ID][]byte) error

	// GetCertifications returns the certifications of the passed tokens.
	// For each token, the callback function is invoked.
	// If a token doesn't have a certification, the function returns an error
	GetCertifications(ids []*token.ID) ([][]byte, error)
}

type TokenDBTransaction interface {
	// GetToken returns the owned tokens and their identifier keys for the passed ids.
	GetToken(ctx context.Context, tokenID token.ID, includeDeleted bool) (*token.Token, []string, error)
	// Delete marks the passed token as deleted by a given identifier (idempotent)
	Delete(ctx context.Context, tokenID token.ID, deletedBy string) error
	// StoreToken stores the passed token record in relation to the passed owner identifiers, if any
	StoreToken(ctx context.Context, tr TokenRecord, owners []string) error
	// SetSpendable updates the spendable flag of the passed token
	SetSpendable(ctx context.Context, tokenID token.ID, spendable bool) error
	// SetSpendableBySupportedTokenFormats sets the spendable flag to true for all the tokens having one of the passed token type.
	// The spendable flag is set to false for the other tokens
	SetSpendableBySupportedTokenFormats(ctx context.Context, formats []token.Format) error
	// Commit commits this transaction
	Commit() error
	// Rollback rollbacks this transaction
	Rollback() error
}

// TokenDB defines a database to store token related info
type TokenDB interface {
	CertificationDB
	// DeleteTokens marks the passsed tokens as deleted
	DeleteTokens(deletedBy string, toDelete ...*token.ID) error
	// IsMine return true if the passed token was stored before
	IsMine(txID string, index uint64) (bool, error)
	// UnspentTokensIterator returns an iterator over all owned tokens
	UnspentTokensIterator() (driver.UnspentTokensIterator, error)
	// LedgerTokensIteratorBy returns an iterator over all unspent ledger tokens
	UnspentLedgerTokensIteratorBy(ctx context.Context) (driver.LedgerTokensIterator, error)
	// UnspentTokensIteratorBy returns an iterator over all tokens owned by the passed wallet identifier and of a given type
	UnspentTokensIteratorBy(ctx context.Context, walletID string, tokenType token.Type) (driver.UnspentTokensIterator, error)
	// SpendableTokensIteratorBy returns an iterator over all tokens owned solely by the passed wallet identifier and of a given type
	SpendableTokensIteratorBy(ctx context.Context, walletID string, typ token.Type) (driver.SpendableTokensIterator, error)
	// UnsupportedTokensIteratorBy returns the minimum information for upgrade about the tokens that are not supported
	UnsupportedTokensIteratorBy(ctx context.Context, walletID string, tokenType token.Type) (driver.UnsupportedTokensIterator, error)
	// ListUnspentTokensBy returns the list of all tokens owned by the passed identifier of a given type
	ListUnspentTokensBy(walletID string, typ token.Type) (*token.UnspentTokens, error)
	// ListUnspentTokens returns the list of all owned tokens
	ListUnspentTokens() (*token.UnspentTokens, error)
	// ListAuditTokens returns the audited tokens for the passed ids
	ListAuditTokens(ids ...*token.ID) ([]*token.Token, error)
	// ListHistoryIssuedTokens returns the list of all issued tokens
	ListHistoryIssuedTokens() (*token.IssuedTokens, error)
	// GetTokenOutputs returns the value of the tokens as they appear on the ledger for the passed ids.
	// For each token, the call-back function is invoked. The call-back function is invoked respecting the order of the passed ids.
	GetTokenOutputs(ids []*token.ID, callback driver.QueryCallbackFunc) error
	// GetTokenMetadata returns the metadata of the tokens for the passed ids.
	GetTokenMetadata(ids []*token.ID) ([][]byte, error)
	// GetTokenOutputsAndMeta retrieves both the token output, metadata, and type for the passed ids.
	GetTokenOutputsAndMeta(ctx context.Context, ids []*token.ID) ([][]byte, [][]byte, []token.Format, error)
	// GetTokens returns the owned tokens and their identifier keys for the passed ids.
	GetTokens(inputs ...*token.ID) ([]*token.Token, error)
	// WhoDeletedTokens for each id, the function return if it was deleted and by who as per the Delete function
	WhoDeletedTokens(inputs ...*token.ID) ([]string, []bool, error)
	// TransactionExists returns true if a token with that transaction id exists in the db
	TransactionExists(ctx context.Context, id string) (bool, error)
	// StorePublicParams stores the public parameters.
	// If they already exist, the function return with no error. No changes are applied.
	StorePublicParams(raw []byte) error
	// PublicParams returns the stored public parameters.
	// If not public parameters are available, it returns nil with no error
	PublicParams() ([]byte, error)
	// PublicParamsByHash returns the public parameters whose hash matches the passed one.
	// If not public parameters are available for that hash, it returns an error
	PublicParamsByHash(rawHash driver.PPHash) ([]byte, error)
	// NewTokenDBTransaction returns a new Transaction to commit atomically multiple operations
	NewTokenDBTransaction() (TokenDBTransaction, error)
	// QueryTokenDetails provides detailed information about tokens
	QueryTokenDetails(params QueryTokenDetailsParams) ([]TokenDetails, error)
	// Balance returns the sun of the amounts of the tokens with type and EID equal to those passed as arguments.
	Balance(ownerEID string, typ token.Type) (uint64, error)
	// SetSupportedTokenFormats sets the supported token formats
	SetSupportedTokenFormats(formats []token.Format) error
}

// TokenDBDriver is the interface for a token database driver
type TokenDBDriver interface {
	// Open opens a token database
	Open(cp ConfigProvider, tmsID token2.TMSID) (TokenDB, error)
}

// TokenNotifier is the observable version of TokenDB
type TokenNotifier driver2.Notifier

// TokenNotifierDriver is the interface for a token database driver
type TokenNotifierDriver interface {
	// Open opens a token database with its listeners
	Open(cp ConfigProvider, tmsID token2.TMSID) (TokenNotifier, error)
}

// TokenLockDB enforces that a token be used only by one process
// A housekeeping job can clean up expired locks (e.g. created_at is more than 5 minutes ago) in order to:
// - avoid that the table grows infinitely
// - unlock tokens that were locked by a process that exited unexpectedly
type TokenLockDB interface {
	// Lock locks a specific token for the consumer TX
	Lock(tokenID *token.ID, consumerTxID transaction.ID) error
	// UnlockByTxID unlocks all tokens locked by the consumer TX
	UnlockByTxID(consumerTxID transaction.ID) error
	// Cleanup removes the locks such that either:
	// 1. The transaction that locked that token is valid or invalid;
	// 2. The lock is too old.
	Cleanup(leaseExpiry time.Duration) error
	// Close closes the database
	Close() error
}

// TokenLockDBDriver is the interface for a token database driver
type TokenLockDBDriver interface {
	// Open opens a token database
	Open(cp ConfigProvider, tmsID token2.TMSID) (TokenLockDB, error)
}

var (
	ErrTokenDoesNotExist = errors.New("token does not exist")
)
