/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package boolpolicy

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
)

// RequestRecipientIdentity requests a policy identity from the given parties.
// The policy expression (e.g. "$0 OR $1") governs how component signatures are evaluated.
func RequestRecipientIdentity(context view.Context, policy string, parties []token.Identity, opts ...token.ServiceOption) (token.Identity, error) {
	return ttx.RequestPolicyIdentity(context, policy, parties, opts...)
}
