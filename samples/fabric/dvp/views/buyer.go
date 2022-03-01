/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package views

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxcc"
)

type BuyHouseView struct{}

func (b *BuyHouseView) Call(context view.Context) (interface{}, error) {
	// 1. Respond to transfer request
	_, _, err := ttxcc.RespondExchangeRecipientIdentities(context)
	assert.NoError(err, "failed getting identity")

	tx, action, err := ttxcc.ReceiveAction(context)
	assert.NoError(err, "failed receiving action")

	err = tx.Transfer(
		ttxcc.MyWalletFromTx(context, tx),
		action.Type, []uint64{action.Amount}, []view.Identity{action.Recipient},
	)
	assert.NoError(err, "failed appending transfer")

	_, err = context.RunView(ttxcc.NewCollectActionsResponderView(tx, action))
	assert.NoError(err, "failed responding to action collect")

	// 2. Endorse Sell Transaction
	// Respond to a request for an identity
	_, err = ttxcc.RespondRequestRecipientIdentity(context)
	assert.NoError(err, "failed to respond to identity request")

	tx, err = ttxcc.ReceiveTransaction(context)
	assert.NoError(err, "failed responding to action collect")

	// 3. Validate transaction.
	// The transaction must contain two actions: One action relation to the transfer of cash,
	// and one action relation to the transfer of the house.

	// 4. Sign and send back
	_, err = context.RunView(ttxcc.NewEndorseView(tx))
	assert.NoError(err, "failed to endorse transaction")

	// 5. Wait for confirmation
	return context.RunView(ttxcc.NewFinalityView(tx))
}
