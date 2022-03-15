/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nftcc

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxcc"
)

func RequestRecipientIdentity(context view.Context, recipient view.Identity, opts ...token.ServiceOption) (view.Identity, error) {
	return ttxcc.RequestRecipientIdentity(context, recipient, opts...)
}

func RespondRequestRecipientIdentity(context view.Context) (view.Identity, error) {
	return ttxcc.RespondRequestRecipientIdentity(context)
}
