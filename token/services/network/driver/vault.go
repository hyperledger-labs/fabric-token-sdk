/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type ValidationCode int

const (
	_               ValidationCode = iota
	Valid                          // Transaction is valid and committed
	Invalid                        // Transaction is invalid and has been discarded
	Busy                           // Transaction does not yet have a validity state
	Unknown                        // Transaction is unknown
	HasDependencies                // Transaction is unknown but has known dependencies
)

type UnspentTokensIterator interface {
	Close()
	Next() (*token.UnspentToken, error)
}

// Vault models the vault service
type Vault interface {
	// GetLastTxID returns the last transaction ID committed into the vault
	GetLastTxID() (string, error)

	// UnspentTokensIteratorBy returns an iterator over all unspent tokens by type and id
	UnspentTokensIteratorBy(id, typ string) (UnspentTokensIterator, error)

	// UnspentTokensIterator returns an iterator over all unspent tokens
	UnspentTokensIterator() (UnspentTokensIterator, error)

	// ListUnspentTokens returns the list of all unspent tokens
	ListUnspentTokens() (*token.UnspentTokens, error)

	// Exists returns true if the token exists in the vault
	Exists(id *token.ID) bool

	// Store the passed token certifications, if applicable
	Store(certifications map[*token.ID][]byte) error

	// TokenVault returns the token vault
	TokenVault() *vault.Vault

	// Status returns the status of the transaction
	Status(id string) (ValidationCode, error)

	// DiscardTx discards the transaction with the passed id
	DiscardTx(id string) error
}
