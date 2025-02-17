/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"context"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type QueryCallbackFunc func(*token.ID, []byte) error

type QueryCallback2Func func(*token.ID, string, []byte, []byte) error

// TxStatus is the status of a transaction
type TxStatus = int

const (
	// Unknown is the status of a transaction that is unknown
	Unknown TxStatus = iota
	// Pending is the status of a transaction that has been submitted to the ledger
	Pending
	// Confirmed is the status of a transaction that has been confirmed by the ledger
	Confirmed
	// Deleted is the status of a transaction that has been deleted due to a failure to commit
	Deleted
)

//go:generate counterfeiter -o mock/uti.go -fake-name UnspentTokensIterator . UnspentTokensIterator

type (
	UnspentTokensIterator     = collections.Iterator[*token.UnspentToken]
	SpendableTokensIterator   = collections.Iterator[*token.UnspentTokenInWallet]
	UnsupportedTokensIterator = collections.Iterator[*token.LedgerToken]
)

type Vault interface {
	QueryEngine() QueryEngine
	CertificationStorage() CertificationStorage
}

//go:generate counterfeiter -o mock/certification_storage.go -fake-name CertificationStorage . CertificationStorage

type CertificationStorage interface {
	Exists(id *token.ID) bool
	Store(certifications map[*token.ID][]byte) error
}

//go:generate counterfeiter -o mock/qe.go -fake-name QueryEngine . QueryEngine

type QueryEngine interface {
	// IsPending returns true if the transaction the passed id refers to is still pending, false otherwise
	IsPending(id *token.ID) (bool, error)
	// GetStatus returns the status of the passed transaction
	GetStatus(txID string) (TxStatus, string, error)
	// IsMine returns true if the passed id is owned by any known wallet
	IsMine(id *token.ID) (bool, error)
	// UnspentTokensIterator returns an iterator over all unspent tokens
	UnspentTokensIterator() (UnspentTokensIterator, error)
	// UnspentTokensIteratorBy returns an iterator of unspent tokens owned by the passed id and whose type is the passed on.
	// The token type can be empty. In that case, tokens of any type are returned.
	UnspentTokensIteratorBy(ctx context.Context, walletID string, tokenType token.Type) (UnspentTokensIterator, error)
	// ListUnspentTokens returns the list of unspent tokens
	ListUnspentTokens() (*token.UnspentTokens, error)
	// ListAuditTokens returns the audited tokens associated to the passed ids
	ListAuditTokens(ids ...*token.ID) ([]*token.Token, error)
	// ListHistoryIssuedTokens returns the list of issues tokens
	ListHistoryIssuedTokens() (*token.IssuedTokens, error)
	// PublicParams returns the public parameters
	PublicParams() ([]byte, error)
	// GetTokenMetadata retrieves the token information for the passed ids.
	// For each id, the callback is invoked to unmarshal the token information
	GetTokenMetadata(ids []*token.ID) ([][]byte, error)
	// GetTokenOutputs retrieves the token output as stored on the ledger for the passed ids.
	// For each id, the callback is invoked to unmarshal the output
	GetTokenOutputs(ids []*token.ID, callback QueryCallbackFunc) error
	// GetTokenOutputsAndMeta retrieves both the token output and information for the passed ids.
	GetTokenOutputsAndMeta(ctx context.Context, ids []*token.ID) ([][]byte, [][]byte, []token.Format, error)
	// GetTokens returns the list of tokens with their respective vault keys
	GetTokens(inputs ...*token.ID) ([]*token.Token, error)
	// WhoDeletedTokens returns info about who deleted the passed tokens.
	// The bool array is an indicator used to tell if the token at a given position has been deleted or not
	WhoDeletedTokens(inputs ...*token.ID) ([]string, []bool, error)
	// Balance returns the sun of the amounts, with 64 bits of precision, of the tokens with type and EID equal to those passed as arguments.
	Balance(id string, tokenType token.Type) (uint64, error)
}
