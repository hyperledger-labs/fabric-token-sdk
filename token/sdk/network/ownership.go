/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package network

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type Ownership interface {
	IsMine(tms *token.ManagementService, tok *token2.Token) ([]string, bool)
	AmIAnAuditor(tms *token.ManagementService) bool
}

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

func (w *WalletOwnership) AmIAnAuditor(tms *token.ManagementService) bool {
	logger.Debugf("WalletOwnership.AmIAnAuditor...")
	for _, identity := range tms.PublicParametersManager().Auditors() {
		logger.Debugf("WalletOwnership.AmIAnAuditor: identity [%s]", identity.String())
		if tms.WalletManager().AuditorWalletByIdentity(identity) != nil {
			logger.Debugf("WalletOwnership.AmIAnAuditor: identity [%s], yes", identity.String())
			return true
		}
	}
	logger.Debugf("WalletOwnership.AmIAnAuditor: no")
	return false
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

func (o *OwnershipMultiplexer) AmIAnAuditor(tms *token.ManagementService) bool {
	for _, ownership := range o.ownerships {
		yes := ownership.AmIAnAuditor(tms)
		if yes {
			return true
		}
	}
	return false
}
