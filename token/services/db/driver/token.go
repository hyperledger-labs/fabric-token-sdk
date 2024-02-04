/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
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
	// OwnerRaw is the serialization of the owner identity
	OwnerRaw []byte
	// Ledger is the raw token as stored on the ledger
	Ledger []byte
	// LedgerMetadata is the metadata associated to the content of Ledger
	LedgerMetadata []byte
	// Quantity is the number of units of Type carried in the token.
	// It is encoded as a string containing a number in base 16. The string has prefix ``0x''.
	Quantity string
	// Type is the type of token
	Type string
	// Amount is the Quantity converted to decimal
	Amount uint64
}

type CertificationDB interface {
	// ExistsCertification returns true if a certification for the passed token exists,
	// false otherwise
	ExistsCertification(id *token.ID) bool

	// StoreCertifications stores the passed certifications
	StoreCertifications(certifications map[*token.ID][]byte) error

	// GetCertifications returns the certifications of the passed tokens.
	// For each token, the callback function is invoked.
	// If a token doesn't have a certification, the function returns an error
	GetCertifications(ids []*token.ID, callback func(*token.ID, []byte) error) error
}

type TokenDB interface {
	CertificationDB
	// StoreOwnerToken stores the passed owner token record in relation to the passed owner identifiers
	StoreOwnerToken(tr TokenRecord, owners []string) error
	// StoreIssuedToken store the issued token record
	StoreIssuedToken(tr TokenRecord) error
	// StoreAuditToken store the audited token record
	StoreAuditToken(tr TokenRecord) error
	// OwnersOf returns the list of owner of a given token
	OwnersOf(txID string, index uint64) (*token.Token, []string, error)
	// Delete marks the passed token as deleted by a given identifier
	Delete(txID string, index uint64, deletedBy string) error
	// DeleteTokens permanently deletes the passsed tokens
	DeleteTokens(toDelete ...*token.ID) error
	// IsMine return true if the passed token was stored before
	IsMine(txID string, index uint64) (bool, error)
	// UnspentTokensIterator returns an iterator over all owned tokens
	UnspentTokensIterator() (driver.UnspentTokensIterator, error)
	// UnspentTokensIteratorBy returns an iterator over all tokens owned by the passed identifier of a given type
	UnspentTokensIteratorBy(ownerEID, typ string) (driver.UnspentTokensIterator, error)
	// ListUnspentTokensBy returns the list of all tokens owned by the passed identifier of a given type
	ListUnspentTokensBy(ownerEID, typ string) (*token.UnspentTokens, error)
	// ListUnspentTokens returns the list of all owned tokens
	ListUnspentTokens() (*token.UnspentTokens, error)
	// ListAuditTokens returns the audited tokens for the passed ids
	ListAuditTokens(ids ...*token.ID) ([]*token.Token, error)
	// ListHistoryIssuedTokens returns the list of all issued tokens
	ListHistoryIssuedTokens() (*token.IssuedTokens, error)
	// GetTokenOutputs returns the value of the tokens as they appear on the ledger for the passed ids.
	// For each token, the call-back function is invoked. The call-back function is invoked respecting the order of the passed ids.
	GetTokenOutputs(ids []*token.ID, callback driver.QueryCallbackFunc) error
	// GetTokenInfos returns the metadata of the tokens for the passed ids.
	// For each token, the call-back function is invoked. The call-back function is invoked respecting the order of the passed ids.
	GetTokenInfos(ids []*token.ID, callback driver.QueryCallbackFunc) error
	// GetTokenInfoAndOutputs returns both value and metadata of the tokens for the passed ids.
	// For each token, the call-back function is invoked. The call-back function is invoked respecting the order of the passed ids.
	GetTokenInfoAndOutputs(ids []*token.ID, callback driver.QueryCallback2Func) error
	// GetAllTokenInfos returns the token metadata for the passed ids
	GetAllTokenInfos(ids []*token.ID) ([][]byte, error)
	// GetTokens returns the tokens and their identifier keys for the passed ids.
	GetTokens(inputs ...*token.ID) ([]string, []*token.Token, error)
	// WhoDeletedTokens for each id, the function return if it was deleted and by who as per the Delete function
	WhoDeletedTokens(inputs ...*token.ID) ([]string, []bool, error)
	// StorePublicParams stores the public parameters
	StorePublicParams(raw []byte) error
	// GetPublicParams return the stored public parameters
	GetPublicParams() ([]byte, error)
}

// TokenDBDriver is the interface for a token database driver
type TokenDBDriver interface {
	// Open opens a token database
	Open(sp view2.ServiceProvider, tmsID token2.TMSID) (TokenDB, error)
}
