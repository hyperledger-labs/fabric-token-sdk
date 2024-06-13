/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type ValidationCode = int

const (
	_       ValidationCode = iota
	Valid                  // Transaction is valid and committed
	Invalid                // Transaction is invalid and has been discarded
	Busy                   // Transaction does not yet have a validity state
	Unknown                // Transaction is unknown
)

type (
	// UnspentTokensIterator models an iterator of unspent tokens
	UnspentTokensIterator = driver2.UnspentTokensIterator
	QueryEngine           = vault.QueryEngine
	CertificationStorage  = vault.CertificationStorage
)

// TokenVault models the token vault
type TokenVault interface {
	// QueryEngine returns the query engine over this vault
	QueryEngine() QueryEngine

	// CertificationStorage returns the certification storage over this vault
	CertificationStorage() CertificationStorage

	// DeleteTokens delete the passed tokens in the passed namespace
	DeleteTokens(ids ...*token.ID) error
}

type QueryExecutor interface {
	GetState(key string) ([]byte, error)
	Done()
}

// Vault models the vault service
type Vault interface {
	NewQueryExecutor() (QueryExecutor, error)

	// GetLastTxID returns the last transaction ID committed into the vault
	GetLastTxID() (string, error)

	// Status returns the status of the transaction
	Status(id string) (ValidationCode, string, error)

	// DiscardTx discards the transaction with the passed id
	DiscardTx(id string, message string) error
}
