/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package token

import (
	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

var (
	SelectorInsufficientFunds                  = errors.New("insufficient funds")
	SelectorSufficientButLockedFunds           = errors.New("sufficient but partially locked funds")
	SelectorSufficientButNotCertifiedFunds     = errors.New("sufficient but partially not certified")
	SelectorSufficientFundsButConcurrencyIssue = errors.New("sufficient funds but concurrency issue")
)

type OwnerFilter interface {
	Contains(identity view.Identity) bool
}

type Selector interface {
	Select(ownerFilter OwnerFilter, q, tokenType string) ([]*token2.Id, token2.Quantity, error)
}
