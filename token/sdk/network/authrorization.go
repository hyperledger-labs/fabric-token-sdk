/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package network

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

// Authorization is an interface that defines method to check the relation between a token or TMS
// and wallets (owner, auditor, etc.)
type Authorization interface {
	// IsMine returns true if the passed token is owned by an owner wallet in the passed TMS
	IsMine(tms *token.ManagementService, tok *token2.Token) ([]string, bool)
	// AmIAnAuditor return true if the passed TMS contains an auditor wallet for any of the auditor identities
	// defined in the public parameters of the passed TMS.
	AmIAnAuditor(tms *token.ManagementService) bool
}

// TMSAuthorization is an owner wallet-based ownership checker
type TMSAuthorization struct{}

// IsMine returns true if the passed token is owned by an owner wallet in the passed TMS
func (w *TMSAuthorization) IsMine(tms *token.ManagementService, tok *token2.Token) ([]string, bool) {
	id, valid := tms.WalletManager().IsMine(tok)
	if !valid {
		return nil, valid
	}
	return []string{id}, true
}

// AmIAnAuditor return true if the passed TMS contains an auditor wallet for any of the auditor identities
// defined in the public parameters of the passed TMS.
func (w *TMSAuthorization) AmIAnAuditor(tms *token.ManagementService) bool {
	for _, identity := range tms.PublicParametersManager().PublicParameters().Auditors() {
		if tms.WalletManager().AuditorWalletByIdentity(identity) != nil {
			return true
		}
	}
	return false
}

// AuthorizationMultiplexer iterates over multiple authorization checker
type AuthorizationMultiplexer struct {
	ownerships []Authorization
}

// NewAuthorizationMultiplexer returns a new AuthorizationMultiplexer for the passed ownership checkers
func NewAuthorizationMultiplexer(ownerships ...Authorization) *AuthorizationMultiplexer {
	return &AuthorizationMultiplexer{ownerships: ownerships}
}

// IsMine returns true it there exists an authorization checker that returns true
func (o *AuthorizationMultiplexer) IsMine(tms *token.ManagementService, tok *token2.Token) ([]string, bool) {
	for _, ownership := range o.ownerships {
		ids, mine := ownership.IsMine(tms, tok)
		if mine {
			return ids, true
		}
	}
	return nil, false
}

// AmIAnAuditor returns true it there exists an authorization checker that returns true
func (o *AuthorizationMultiplexer) AmIAnAuditor(tms *token.ManagementService) bool {
	for _, ownership := range o.ownerships {
		yes := ownership.AmIAnAuditor(tms)
		if yes {
			return true
		}
	}
	return false
}
