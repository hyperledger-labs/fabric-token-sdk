/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type Authorization interface {
	// IsMine returns true if the passed token is owned by an owner wallet in the passed TMS
	IsMine(tok *token2.Token) ([]string, bool)
	// AmIAnAuditor return true if the passed TMS contains an auditor wallet for any of the auditor identities
	// defined in the public parameters of the passed TMS.
	AmIAnAuditor() bool
	// Issued returns true if the passed issuer issued the passed token
	Issued(issuer token.Identity, tok *token2.Token) bool
}

// WalletBasedAuthorization is a wallet-based authorization implementation
type WalletBasedAuthorization struct {
	PublicParameters driver.PublicParameters
	WalletService    driver.WalletService
}

func NewTMSAuthorization(publicParameters driver.PublicParameters, walletService driver.WalletService) *WalletBasedAuthorization {
	return &WalletBasedAuthorization{PublicParameters: publicParameters, WalletService: walletService}
}

// IsMine returns true if the passed token is owned by an owner wallet in the passed TMS
func (w *WalletBasedAuthorization) IsMine(tok *token2.Token) ([]string, bool) {
	wallet, err := w.WalletService.OwnerWallet(tok.Owner.Raw)
	if err != nil {
		return nil, false
	}
	return []string{wallet.ID()}, true
}

// AmIAnAuditor return true if the passed TMS contains an auditor wallet for any of the auditor identities
// defined in the public parameters of the passed TMS.
func (w *WalletBasedAuthorization) AmIAnAuditor() bool {
	for _, identity := range w.PublicParameters.Auditors() {
		if _, err := w.WalletService.AuditorWallet(identity); err == nil {
			return true
			break
		}
	}
	return false
}

func (w *WalletBasedAuthorization) Issued(issuer token.Identity, tok *token2.Token) bool {
	_, err := w.WalletService.IssuerWallet(issuer)
	return err == nil
}

// AuthorizationMultiplexer iterates over multiple authorization checker
type AuthorizationMultiplexer struct {
	authorizations []Authorization
}

// NewAuthorizationMultiplexer returns a new AuthorizationMultiplexer for the passed ownership checkers
func NewAuthorizationMultiplexer(ownerships ...Authorization) *AuthorizationMultiplexer {
	return &AuthorizationMultiplexer{authorizations: ownerships}
}

// IsMine returns true it there exists an authorization checker that returns true
func (o *AuthorizationMultiplexer) IsMine(tok *token2.Token) ([]string, bool) {
	for _, authorization := range o.authorizations {
		ids, mine := authorization.IsMine(tok)
		if mine {
			return ids, true
		}
	}
	return nil, false
}

// AmIAnAuditor returns true it there exists an authorization checker that returns true
func (o *AuthorizationMultiplexer) AmIAnAuditor() bool {
	for _, authorization := range o.authorizations {
		yes := authorization.AmIAnAuditor()
		if yes {
			return true
		}
	}
	return false
}

func (o *AuthorizationMultiplexer) Issued(issuer token.Identity, tok *token2.Token) bool {
	for _, authorization := range o.authorizations {
		yes := authorization.Issued(issuer, tok)
		if yes {
			return true
		}
	}
	return false
}

// OwnerType returns the type of owner (e.g. 'idemix' or 'htlc') and the identity bytes
func (o *AuthorizationMultiplexer) OwnerType(raw []byte) (string, []byte, error) {
	owner, err := identity.UnmarshalTypedIdentity(raw)
	if err != nil {
		return "", nil, err
	}
	return owner.Type, owner.Identity, nil
}
