/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package processor

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

// WalletOwnership is an owner wallet-based ownership checker
type WalletOwnership struct{}

// IsMine returns true it there exists an owner wallet for the token's owner
func (w *WalletOwnership) IsMine(tms *token.ManagementService, tok *token2.Token) bool {
	return tms.WalletManager().OwnerWalletByIdentity(tok.Owner.Raw) != nil
}

// OwnershipMultiplexer iterates over multiple ownership checker
type OwnershipMultiplexer struct {
	ownerships []Ownership
}

// NewOwnershipMultiplexer returns a new OwnershipMultiplexer for the passed ownership checkers
func NewOwnershipMultiplexer(ownerships ...Ownership) *OwnershipMultiplexer {
	return &OwnershipMultiplexer{ownerships: ownerships}
}

// IsMine returns true it there exists an ownership checker that returns true
func (o *OwnershipMultiplexer) IsMine(tms *token.ManagementService, tok *token2.Token) bool {
	for _, ownership := range o.ownerships {
		if ownership.IsMine(tms, tok) {
			return true
		}
	}
	return false
}
