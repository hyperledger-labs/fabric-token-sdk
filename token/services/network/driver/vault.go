/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

// Vault models the vault service
type Vault interface {
	// GetLastTxID returns the last transaction ID committed into the vault
	GetLastTxID() (string, error)

	// ListUnspentTokens returns the list of all unspent tokens
	ListUnspentTokens() (*token.UnspentTokens, error)

	// Exists returns true if the token exists in the vault
	Exists(id *token.ID) bool

	// Store the passed token certifications, if applicable
	Store(certifications map[*token.ID][]byte) error

	// TokenVault returns the token vault
	TokenVault() *vault.Vault
}
