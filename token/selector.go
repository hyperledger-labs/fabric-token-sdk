/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package token

import (
	"github.com/pkg/errors"

	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

var (
	// SelectorInsufficientFunds is returned when funds are not sufficient to cover the request
	SelectorInsufficientFunds = errors.New("insufficient funds")
	// SelectorSufficientButLockedFunds is returned when funds are sufficient to cover the request, but some tokens are locked
	// by other transactions
	SelectorSufficientButLockedFunds = errors.New("sufficient but partially locked funds")
	// SelectorSufficientButNotCertifiedFunds is returned when funds are sufficient to cover the request, but some tokens
	// are not yet certified and therefore cannot be used.
	SelectorSufficientButNotCertifiedFunds = errors.New("sufficient but partially not certified")
	// SelectorSufficientFundsButConcurrencyIssue is returned when funds are sufficient to cover the request, but
	// concurrency issues does not make some of the selected tokens available.
	SelectorSufficientFundsButConcurrencyIssue = errors.New("sufficient funds but concurrency issue")
)

// OwnerFilter tells if a passed identity is recognized
type OwnerFilter interface {
	ID() string
	// ContainsToken returns true if the passed token is recognized, false otherwise.
	ContainsToken(token *token2.UnspentToken) bool
}

// Selector is the interface of token selectors
type Selector interface {
	// Select returns the list of token identifiers where
	// 1. The owner match the passed owner filter.
	// 2. The type is equal to the passed token type.
	// 3. The sum of amount in each token is at least the passed quantity.
	// Quantity is a string in decimal format
	// Notice that, the quantity selected might exceed the quantity requested due to the amounts
	// stored in each token.
	Select(ownerFilter OwnerFilter, q, tokenType string) ([]*token2.ID, token2.Quantity, error)
}
