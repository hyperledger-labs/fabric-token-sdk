/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package views

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/services/endorser"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/services/state"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
)

type BuyHouseView struct{}

func (b *BuyHouseView) Call(context view.Context) (interface{}, error) {
	// 1. Respond to transfer request
	_, other, err := ttx.ExchangeRecipientIdentitiesResponder(context)
	assert.NoError(err, "failed getting identity")

	tokenTx, action, err := ttx.ReceiveAction(context)
	assert.NoError(err, "failed receiving action")

	err = tokenTx.Transfer(
		ttx.MyWalletForChannel(context, tokenTx.Channel()),
		action.Type, []uint64{action.Amount}, []view.Identity{action.Recipient},
	)
	assert.NoError(err, "failed appending transfer")

	_, err = context.RunView(ttx.NewCollectActionsResponderView(tokenTx, action))
	assert.NoError(err, "failed responding to action collect")

	// 2. Endorse Sell Transaction
	// Respond to a request for an identity
	me, err := state.RespondRequestRecipientIdentity(context)
	assert.NoError(err, "failed to respond to identity request")

	txBoxed, err := context.RunView(endorser.NewReceiveTransactionView())
	assert.NoError(err, "failed responding to action collect")
	tx := txBoxed.(*endorser.Transaction)

	// 3. Validate transaction: The transaction consists of two namespace (zkat and house)
	tokenTx, err = ttx.Wrap(tx)
	assert.NoError(err)

	inputs, err := tokenTx.Inputs()
	assert.NoError(err, "failed getting outputs")
	assert.Equal(1, inputs.Count())
	// TODO: enable this by completing the missing code
	//assert.True(tokenTx.Inputs().Owners().IsEqualToAt(0, me), "invalid sender [%s] not in [%s]", tokenTx.Inputs().String())
	outputs, err := tokenTx.Outputs()
	assert.NoError(err, "failed getting outputs")
	assert.Equal(1, outputs.Count())
	assert.Equal(1, outputs.ByRecipient(other).Count())

	stateTx, err := state.Wrap(tx)
	assert.NoError(err)

	assert.Equal(1, stateTx.Outputs().Count())
	house := &House{}
	assert.NoError(stateTx.Outputs().At(0).State(house))
	assert.Equal(action.Amount, house.Valuation)
	assert.Equal(me, house.Owner)

	// 4. Sign and send back
	_, err = context.RunView(endorser.NewEndorseView(tx, append([]view.Identity{me}, tokenTx.Signers()...)...))
	assert.NoError(err)

	// 5. Wait for confirmation
	return context.RunView(endorser.NewFinalityView(tx))
}
