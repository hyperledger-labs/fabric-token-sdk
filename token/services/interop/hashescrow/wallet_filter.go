/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package hashescrow

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections/iterators"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type SelectFunction = func(*token.UnspentToken, *Script) (bool, error)

type PreImageSelector struct {
	preImage []byte
}

func (f *PreImageSelector) Filter(tok *token.UnspentToken, script *Script) (bool, error) {
	logger.Debugf("token [%s,%s,%s,%s] contains a hash escrow script? Yes", tok.Id, view.Identity(tok.Owner).UniqueID(), tok.Type, tok.Quantity)

	_, image, err := script.ResolveRecipientForPreImage(f.preImage)
	if err != nil {
		logger.Debugf("hash escrow script does not match pre-image [%s]: [%s]", logging.Base64(f.preImage), err)

		return false, nil
	}
	logger.Debugf("hash escrow script matches pre-image [%s] with image [%s]", logging.Base64(f.preImage), logging.Base64(image))

	return true, nil
}

func IsScript(selector SelectFunction) iterators.Predicate[*token.UnspentToken] {
	return func(tok *token.UnspentToken) bool {
		owner, err := identity.UnmarshalTypedIdentity(tok.Owner)
		if err != nil {
			logger.Debugf("Is Mine [%s,%s,%s]? No, failed unmarshalling [%s]", view.Identity(tok.Owner), tok.Type, tok.Quantity, err)

			return false
		}
		if owner.Type != ScriptType {
			return false
		}

		script := &Script{}
		if err := json.Unmarshal(owner.Identity, script); err != nil {
			logger.Debugf("token [%s,%s,%s,%s] contains a hash escrow script? No", tok.Id, view.Identity(tok.Owner).UniqueID(), tok.Type, tok.Quantity)

			return false
		}
		if err := script.Validate(); err != nil {
			logger.Debugf("token [%s,%s,%s,%s] contains an invalid hash escrow script: [%s]", tok.Id, view.Identity(tok.Owner).UniqueID(), tok.Type, tok.Quantity, err)

			return false
		}

		pickItem, err := selector(tok, script)
		if err != nil {
			logger.Errorf("failed to select (token,script)[%v:%v] pair: %w", tok, script, err)

			return false
		}

		return pickItem
	}
}
