/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package multisig

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/multisig"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	token3 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

var logger = logging.MustGetLogger("token-sdk.services.multisig")

// EscrowAuth implements the Authorization interface for this script
type EscrowAuth struct {
	WalletService driver.WalletService
}

func NewEscrowAuth(walletService driver.WalletService) *EscrowAuth {
	return &EscrowAuth{WalletService: walletService}
}

// AmIAnAuditor returns false for script ownership
func (s *EscrowAuth) AmIAnAuditor() bool {
	return false
}

// IsMine returns true if either the sender or the recipient is in one of the owner wallets.
// It returns an empty wallet id.
func (s *EscrowAuth) IsMine(tok *token3.Token) (string, []string, bool) {
	owner, err := identity.UnmarshalTypedIdentity(tok.Owner)
	if err != nil {
		logger.Debugf("Is Mine [%s,%s,%s]? No, failed unmarshalling [%s]", view.Identity(tok.Owner), tok.Type, tok.Quantity, err)
		return "", nil, false
	}
	if owner.Type != multisig.Multisig {
		logger.Debugf("Is Mine [%s,%s,%s]? No, owner type is [%s] instead of [%s]", view.Identity(tok.Owner), tok.Type, tok.Quantity, owner.Type, multisig.Multisig)
		return "", nil, false
	}
	escrow := &multisig.MultiIdentity{}
	if err := escrow.Deserialize(owner.Identity); err != nil {
		logger.Debugf("Is Mine [%s,%s,%s]? No, failed unmarshalling [%s]", view.Identity(tok.Owner), tok.Type, tok.Quantity, err)
		return "", nil, false
	}
	var ids []string
	for i := 0; i < len(escrow.Identities); i++ {
		logger.Debugf("Is Mine [%s,%s,%s] as an escrow co-owner?", view.Identity(tok.Owner), tok.Type, tok.Quantity)
		if wallet, err := s.WalletService.OwnerWallet(escrow.Identities[i]); err == nil {
			logger.Debugf("Is Mine [%s,%s,%s] as an escrow co-owner? Yes", view.Identity(tok.Owner), tok.Type, tok.Quantity)
			ids = append(ids, escrowWallet(wallet))
		}
	}
	logger.Debugf("Is Mine [%s,%s,%s]? %b", len(ids) != 0, view.Identity(tok.Owner), tok.Type, tok.Quantity)
	return "", ids, len(ids) != 0
}

func (s *EscrowAuth) Issued(issuer driver.Identity, tok *token3.Token) bool {
	return false
}

func (s *EscrowAuth) OwnerType(raw []byte) (string, []byte, error) {
	owner, err := identity.UnmarshalTypedIdentity(raw)
	if err != nil {
		return "", nil, err
	}
	return owner.Type, owner.Identity, nil
}

type ownerWallet interface {
	ID() string
}

func escrowWallet(w ownerWallet) string {
	return "escrow" + w.ID()
}
