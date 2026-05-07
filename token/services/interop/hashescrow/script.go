/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package hashescrow

import (
	"context"
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/htlc"
	token3 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

// HashInfo contains hash configuration used by hash-based escrow scripts.
// Reuse the HTLC hash representation to keep hashing semantics identical.
type HashInfo = htlc.HashInfo

// Script contains the details of a hash-based escrow lock.
// Either sender or recipient can claim by presenting a valid preimage and signature.
type Script struct {
	Sender    view.Identity
	Recipient view.Identity
	HashInfo  HashInfo
}

// Validate performs the following checks:
// - sender must be set
// - recipient must be set
// - hash info must be valid
func (s *Script) Validate() error {
	if s.Sender.IsNone() {
		return errors.New("sender not set")
	}
	if s.Recipient.IsNone() {
		return errors.New("recipient not set")
	}
	if err := s.HashInfo.Validate(); err != nil {
		return err
	}

	return nil
}

func (s *Script) FromBytes(raw []byte) error {
	return json.Unmarshal(raw, s)
}

// ScriptAuth implements token ownership checks for hash-based escrow scripts.
type ScriptAuth struct {
	WalletService driver.WalletService
}

func NewScriptAuth(walletService driver.WalletService) *ScriptAuth {
	return &ScriptAuth{WalletService: walletService}
}

// AmIAnAuditor returns false for script ownership.
func (s *ScriptAuth) AmIAnAuditor() bool {
	return false
}

// IsMine returns true if either the sender or recipient belongs to one of our owner wallets.
func (s *ScriptAuth) IsMine(ctx context.Context, tok *token3.Token) (string, []string, bool) {
	owner, err := identity.UnmarshalTypedIdentity(tok.Owner)
	if err != nil {
		logger.DebugfContext(ctx, "Is Mine [%s,%s,%s]? No, failed unmarshalling [%s]", view.Identity(tok.Owner), tok.Type, tok.Quantity, err)

		return "", nil, false
	}
	if owner.Type != ScriptType {
		logger.DebugfContext(ctx, "Is Mine [%s,%s,%s]? No, owner type is [%s] instead of [%s]", view.Identity(tok.Owner), tok.Type, tok.Quantity, owner.Type, ScriptType)

		return "", nil, false
	}
	script := &Script{}
	if err := json.Unmarshal(owner.Identity, script); err != nil {
		logger.DebugfContext(ctx, "Is Mine [%s,%s,%s]? No, failed unmarshalling [%s]", view.Identity(tok.Owner), tok.Type, tok.Quantity, err)

		return "", nil, false
	}
	if script.Sender.IsNone() || script.Recipient.IsNone() {
		logger.DebugfContext(ctx, "Is Mine [%s,%s,%s]? No, invalid content [%v]", view.Identity(tok.Owner), tok.Type, tok.Quantity, script)

		return "", nil, false
	}

	var ids []string

	logger.DebugfContext(ctx, "Is Mine [%s,%s,%s] as sender?", view.Identity(tok.Owner), tok.Type, tok.Quantity)
	if wallet, err := s.WalletService.OwnerWallet(ctx, script.Sender); err == nil {
		ids = append(ids, senderWallet(wallet))
	}

	logger.DebugfContext(ctx, "Is Mine [%s,%s,%s] as recipient?", view.Identity(tok.Owner), tok.Type, tok.Quantity)
	if wallet, err := s.WalletService.OwnerWallet(ctx, script.Recipient); err == nil {
		ids = append(ids, recipientWallet(wallet))
	}

	return "", ids, len(ids) != 0
}

func (s *ScriptAuth) Issued(ctx context.Context, issuer driver.Identity, tok *token3.Token) bool {
	return false
}

func (s *ScriptAuth) OwnerType(raw []byte) (driver.IdentityType, []byte, error) {
	owner, err := identity.UnmarshalTypedIdentity(raw)
	if err != nil {
		return driver.ZeroIdentityType, nil, err
	}

	return owner.Type, owner.Identity, nil
}

type ownerWallet interface {
	ID() string
}

func senderWallet(w ownerWallet) string {
	return "hashescrow.sender" + w.ID()
}

func recipientWallet(w ownerWallet) string {
	return "hashescrow.recipient" + w.ID()
}
