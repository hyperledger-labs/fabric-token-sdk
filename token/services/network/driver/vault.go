/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
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

type TxStatus = int

type (
	// UnspentTokensIterator models an iterator of unspent tokens
	UnspentTokensIterator = driver2.UnspentTokensIterator
	QueryEngine           = driver2.QueryEngine
	CertificationStorage  = driver2.CertificationStorage
)

// TokenVault models the token vault
type TokenVault interface {
	driver2.Vault

	// DeleteTokens delete the passed tokens in the passed namespace
	DeleteTokens(ids ...*token.ID) error
}

type TokenVaultProvider interface {
	Vault(network, channel, namespace string) (TokenVault, error)
}

// Vault models the vault service
type Vault interface {
	// Status returns the status of the transaction
	Status(id string) (ValidationCode, string, error)

	// DiscardTx discards the transaction with the passed id
	DiscardTx(id string, message string) error
}
