/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package multisig

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
)

// RequestRecipientIdentity requests the recipient identity for the given parties.
// It returns a multisig identity. All the parties are notified about the participants in the multisig identity.
func RequestRecipientIdentity(context view.Context, parties []token.Identity, opts ...token.ServiceOption) (token.Identity, error) {
	return ttx.RequestMultisigIdentity(context, parties, opts...)
}
