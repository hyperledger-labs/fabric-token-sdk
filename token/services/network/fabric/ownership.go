/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabric

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

// WalletOwnership is an owner wallet-based ownership checker
type WalletOwnership struct{}

// IsMine returns true it there exists an owner wallet for the token's owner
func (w *WalletOwnership) IsMine(tms *token.ManagementService, tok *token2.Token) ([]string, bool) {
	wallet := tms.WalletManager().OwnerWalletByIdentity(tok.Owner.Raw)
	if wallet == nil {
		return nil, false
	}
	return []string{wallet.ID()}, true
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
func (o *OwnershipMultiplexer) IsMine(tms *token.ManagementService, tok *token2.Token) ([]string, bool) {
	for _, ownership := range o.ownerships {
		ids, mine := ownership.IsMine(tms, tok)
		if mine {
			return ids, true
		}
	}
	return nil, false
}
