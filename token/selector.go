/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package token

import (
	"context"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

var (
	// ErrSelectorInsufficientFunds is returned when funds are not sufficient to cover the request
	ErrSelectorInsufficientFunds = errors.New("insufficient funds")
	// ErrSelectorSufficientButLockedFunds is returned when funds are sufficient to cover the request, but some tokens are locked
	// by other transactions
	ErrSelectorSufficientButLockedFunds = errors.New("sufficient but partially locked funds")
	// ErrSelectorSufficientButNotCertifiedFunds is returned when funds are sufficient to cover the request, but some tokens
	// are not yet certified and therefore cannot be used.
	ErrSelectorSufficientButNotCertifiedFunds = errors.New("sufficient but partially not certified")
	// ErrSelectorSufficientFundsButConcurrencyIssue is returned when funds are sufficient to cover the request, but
	// concurrency issues does not make some of the selected tokens available.
	ErrSelectorSufficientFundsButConcurrencyIssue = errors.New("sufficient funds but concurrency issue")
)

// OwnerFilter tells if a passed identity is recognized
type OwnerFilter interface {
	// ID is the wallet identifier of the owner
	ID() string
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
	Select(ctx context.Context, ownerFilter OwnerFilter, q string, tokenType token.Type) ([]*token.ID, token.Quantity, error)
	// Close closes the selector and releases its memory/cpu resources
	Close() error
}
