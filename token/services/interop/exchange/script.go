/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package exchange

import (
	"crypto"
	"encoding/json"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/encoding"
	token3 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

// HashInfo contains the information regarding the hashing
type HashInfo struct {
	Hash         []byte
	HashFunc     crypto.Hash
	HashEncoding encoding.Encoding
}

// Script contains the details of an exchange
type Script struct {
	Sender    view.Identity
	Recipient view.Identity
	Deadline  time.Time
	HashInfo  HashInfo
}

// ScriptOwnership implements the Ownership interface for scripts
type ScriptOwnership struct{}

// AmIAnAuditor returns false for script ownership
func (s *ScriptOwnership) AmIAnAuditor(tms *token.ManagementService) bool {
	return false
}

// IsMine returns true if one is either a sender or a recipient of an exchange script
func (s *ScriptOwnership) IsMine(tms *token.ManagementService, tok *token3.Token) ([]string, bool) {
	owner, err := identity.UnmarshallRawOwner(tok.Owner.Raw)
	if err != nil {
		logger.Debugf("Is Mine [%s,%s,%s]? No, failed unmarshalling [%s]", view.Identity(tok.Owner.Raw), tok.Type, tok.Quantity, err)
		return nil, false
	}
	if owner.Type != ScriptTypeExchange {
		logger.Debugf("Is Mine [%s,%s,%s]? No, owner type is [%s] instead of [%s]", view.Identity(tok.Owner.Raw), tok.Type, tok.Quantity, owner.Type, ScriptTypeExchange)
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

	// I'm either the sender
	logger.Debugf("Is Mine [%s,%s,%s] as a sender?", view.Identity(tok.Owner.Raw), tok.Type, tok.Quantity)
	if tms.WalletManager().OwnerWalletByIdentity(script.Sender) != nil {
		logger.Debugf("Is Mine [%s,%s,%s] as a sender? Yes", view.Identity(tok.Owner.Raw), tok.Type, tok.Quantity)
		return nil, true
	}

	// or the recipient
	logger.Debugf("Is Mine [%s,%s,%s] as a recipient?", view.Identity(tok.Owner.Raw), tok.Type, tok.Quantity)
	if tms.WalletManager().OwnerWalletByIdentity(script.Recipient) != nil {
		logger.Debugf("Is Mine [%s,%s,%s] as a recipient? Yes", view.Identity(tok.Owner.Raw), tok.Type, tok.Quantity)
		return nil, true
	}

	logger.Debugf("Is Mine [%s,%s,%s]? No", view.Identity(tok.Owner.Raw), tok.Type, tok.Quantity)

	return nil, false
}
