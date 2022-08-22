/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package htlc

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
)

// ExchangeRecipientIdentities executes the ttx ExchangeRecipientIdentitiesView
func ExchangeRecipientIdentities(context view.Context, walletID string, recipient view.Identity, opts ...token.ServiceOption) (view.Identity, view.Identity, error) {
	return ttx.ExchangeRecipientIdentities(context, walletID, recipient, opts...)
}

// RespondExchangeRecipientIdentities executes the ttx RespondExchangeRecipientIdentitiesView
func RespondExchangeRecipientIdentities(context view.Context) (view.Identity, view.Identity, error) {
	return ttx.RespondExchangeRecipientIdentities(context)
}
