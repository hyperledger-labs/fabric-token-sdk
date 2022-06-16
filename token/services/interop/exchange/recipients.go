/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package exchange

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
)

// RequestRecipientIdentity executes the RequestRecipientIdentityView.
// The sender contacts the recipient's FSC node identified via the passed view identity.
// The sender gets back the identity the recipient wants to use to assign ownership of tokens.
func RequestRecipientIdentity(context view.Context, recipient view.Identity, opts ...token.ServiceOption) (view.Identity, error) {
	return ttx.RequestRecipientIdentity(context, recipient, opts...)
}

// RespondRequestRecipientIdentity executes the RespondRequestRecipientIdentityView.
// The recipient sends back the identity to receive ownership of tokens.
// The identity is taken from the default wallet
func RespondRequestRecipientIdentity(context view.Context) (view.Identity, error) {
	return ttx.RespondRequestRecipientIdentity(context)
}

// ExchangeRecipientIdentities executes the ExchangeRecipientIdentitiesView using by passed wallet id to
// derive the recipient identity to send to the passed recipient.
// The function returns, the recipient identity of the sender, the recipient identity of the recipient
func ExchangeRecipientIdentities(context view.Context, walletID string, recipient view.Identity, opts ...token.ServiceOption) (view.Identity, view.Identity, error) {
	return ttx.ExchangeRecipientIdentities(context, walletID, recipient, opts...)
}

func RespondExchangeRecipientIdentities(context view.Context) (view.Identity, view.Identity, error) {
	return ttx.RespondExchangeRecipientIdentities(context)
}
