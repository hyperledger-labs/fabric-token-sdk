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
	// Namespace is the namespace where the token was created
	Namespace string
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
	StoreOwnerToken(tr TokenRecord, owners []string) error
	StoreIssuedToken(tr TokenRecord) error
	StoreAuditToken(tr TokenRecord) error
	OwnersOf(ns, txID string, index uint64) (*token.Token, []string, error)
	Delete(ns, txID string, index uint64, deletedBy string) error
	DeleteTokens(ns string, ids ...*token.ID) error
	IsMine(ns, txID string, index uint64) (bool, error)
	UnspentTokensIterator(ns string) (driver.UnspentTokensIterator, error)
	UnspentTokensIteratorBy(ns, ownerEID, typ string) (driver.UnspentTokensIterator, error)
	ListUnspentTokensBy(ns, ownerEID, typ string) (*token.UnspentTokens, error)
	ListUnspentTokens(ns string) (*token.UnspentTokens, error)
	ListAuditTokens(ns string, ids ...*token.ID) ([]*token.Token, error)
	ListHistoryIssuedTokens(ns string) (*token.IssuedTokens, error)
	GetTokenOutputs(ns string, ids []*token.ID, callback driver.QueryCallbackFunc) error
	GetTokenInfos(ns string, ids []*token.ID, callback driver.QueryCallbackFunc) error
	GetTokenInfoAndOutputs(ns string, ids []*token.ID, callback driver.QueryCallback2Func) error
	GetAllTokenInfos(ns string, ids []*token.ID) ([][]byte, error)
	GetTokens(ns string, inputs ...*token.ID) ([]string, []*token.Token, error)
	WhoDeletedTokens(ns string, inputs ...*token.ID) ([]string, []bool, error)
	StorePublicParams(raw []byte) error
	GetRawPublicParams() ([]byte, error)
}

// Driver is the interface for a database driver
type Driver interface {
	// Open opens a database connection
	Open(sp view2.ServiceProvider, tmsID token2.TMSID) (TokenDB, error)
}
