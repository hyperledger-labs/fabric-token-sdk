/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tokens

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type Issued interface {
	// Issued returns true if the passed issuer issued the passed token
	Issued(tms *token.ManagementService, issuer token.Identity, tok *token2.Token) bool
}

// WalletIssued is an owner wallet-based Issued checker
type WalletIssued struct{}

// Issued returns true if the passed issuer issued the passed token
func (w *WalletIssued) Issued(tms *token.ManagementService, issuer token.Identity, tok *token2.Token) bool {
	return tms.WalletManager().IssuerWallet(issuer) != nil
}

// IssuedMultiplexer iterates over multiple Issued checker
type IssuedMultiplexer struct {
	Checkers []Issued
}

// NewIssuedMultiplexer returns a new IssuedMultiplexer for the passed Issued checkers
func NewIssuedMultiplexer(checkers ...Issued) *IssuedMultiplexer {
	return &IssuedMultiplexer{Checkers: checkers}
}

// Issued returns true if the passed issuer issued the passed token
func (o *IssuedMultiplexer) Issued(tms *token.ManagementService, issuer token.Identity, tok *token2.Token) bool {
	for _, Issued := range o.Checkers {
		if Issued.Issued(tms, issuer, tok) {
			return true
		}
	}
	return false
}
