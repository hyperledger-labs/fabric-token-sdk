/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package htlc

import (
	"crypto"
	"encoding/json"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/encoding"
	token3 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

// HashInfo contains the information regarding the hashing
type HashInfo struct {
	Hash         []byte
	HashFunc     crypto.Hash
	HashEncoding encoding.Encoding
}

// Validate checks that the hash and encoding functions are available
func (i *HashInfo) Validate() error {
	if !i.HashFunc.Available() {
		return errors.New("hash function not available")
	}
	if !i.HashEncoding.Available() {
		return errors.New("encoding function not available")
	}
	return nil
}

// Image computes the image of the passed pre-image using the hash and encoding function of this struct
func (i *HashInfo) Image(preImage []byte) ([]byte, error) {
	if err := i.Validate(); err != nil {
		return nil, errors.WithMessagef(err, "hash info not valid")
	}
	hash := i.HashFunc.New()
	if _, err := hash.Write(preImage); err != nil {
		return nil, errors.Wrapf(err, "failed to compute hash image")
	}
	image := hash.Sum(nil)
	image = []byte(i.HashEncoding.New().EncodeToString(image))
	return image, nil
}

// Script contains the details of an htlc
type Script struct {
	Sender    view.Identity
	Recipient view.Identity
	Deadline  time.Time
	HashInfo  HashInfo
}

// Validate performs the following checks:
// - The sender must be set
// - The recipient must be set
// - The deadline must be after the passed time reference
// - HashInfo must be Available
func (s *Script) Validate(timeReference time.Time) error {
	if s.Sender.IsNone() {
		return errors.New("sender not set")
	}
	if s.Recipient.IsNone() {
		return errors.New("recipient not set")
	}
	if s.Deadline.Before(timeReference) {
		return errors.New("expiration date has already passed")
	}
	if err := s.HashInfo.Validate(); err != nil {
		return err
	}
	return nil
}

// ScriptOwnership implements the Ownership interface for scripts
type ScriptOwnership struct{}

// AmIAnAuditor returns false for script ownership
func (s *ScriptOwnership) AmIAnAuditor(tms *token.ManagementService) bool {
	return false
}

// IsMine returns true if one is either a sender or a recipient of an htlc script
func (s *ScriptOwnership) IsMine(tms *token.ManagementService, tok *token3.Token) ([]string, bool) {
	owner, err := identity.UnmarshallRawOwner(tok.Owner.Raw)
	if err != nil {
		logger.Debugf("Is Mine [%s,%s,%s]? No, failed unmarshalling [%s]", view.Identity(tok.Owner.Raw), tok.Type, tok.Quantity, err)
		return nil, false
	}
	if owner.Type != ScriptType {
		logger.Debugf("Is Mine [%s,%s,%s]? No, owner type is [%s] instead of [%s]", view.Identity(tok.Owner.Raw), tok.Type, tok.Quantity, owner.Type, ScriptType)
	}
	script := &Script{}
	if err := json.Unmarshal(owner.Identity, script); err != nil {
		logger.Debugf("Is Mine [%s,%s,%s]? No, failed unmarshalling [%s]", view.Identity(tok.Owner.Raw), tok.Type, tok.Quantity, err)
		return nil, false
	}
	if script.Sender.IsNone() || script.Recipient.IsNone() {
		logger.Debugf("Is Mine [%s,%s,%s]? No, invalid content [%v]", view.Identity(tok.Owner.Raw), tok.Type, tok.Quantity, script)
		return nil, false
	}

	var ids []string
	// I'm either the sender
	logger.Debugf("Is Mine [%s,%s,%s] as a sender?", view.Identity(tok.Owner.Raw), tok.Type, tok.Quantity)
	if wallet := tms.WalletManager().OwnerWalletByIdentity(script.Sender); wallet != nil {
		logger.Debugf("Is Mine [%s,%s,%s] as a sender? Yes", view.Identity(tok.Owner.Raw), tok.Type, tok.Quantity)
		ids = append(ids, senderWallet(wallet))
	}

	// or the recipient
	logger.Debugf("Is Mine [%s,%s,%s] as a recipient?", view.Identity(tok.Owner.Raw), tok.Type, tok.Quantity)
	if wallet := tms.WalletManager().OwnerWalletByIdentity(script.Recipient); wallet != nil {
		logger.Debugf("Is Mine [%s,%s,%s] as a recipient? Yes", view.Identity(tok.Owner.Raw), tok.Type, tok.Quantity)
		ids = append(ids, recipientWallet(wallet))
	}

	logger.Debugf("Is Mine [%s,%s,%s]? %b", len(ids) != 0, view.Identity(tok.Owner.Raw), tok.Type, tok.Quantity)
	return ids, len(ids) != 0
}

func senderWallet(w *token.OwnerWallet) string {
	return "htlc.sender" + w.ID()
}

func recipientWallet(w *token.OwnerWallet) string {
	return "htlc.recipient" + w.ID()
}
