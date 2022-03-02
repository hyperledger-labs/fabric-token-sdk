/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package views

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/tcc/dvp/views/house"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/nftcc"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxcc"
)

type BuyHouseView struct{}

func (b *BuyHouseView) Call(context view.Context) (interface{}, error) {
	// Respond to a request for an identity to transfer the house
	meHouse, err := nftcc.RespondRequestRecipientIdentity(context)
	assert.NoError(err, "failed to respond to identity request")

	// Respond to a request to exchange identifies for the cash transfer
	meCash, otherCash, err := ttxcc.RespondExchangeRecipientIdentities(context)
	assert.NoError(err, "failed getting identity")

	// Receive the request to transfer action
	tx, action, err := ttxcc.ReceiveAction(context)
	assert.NoError(err, "failed receiving action")

	// check transaction, it must contain the house transfer
	nfttx := nftcc.Wrap(tx)
	outputs, err := nfttx.Outputs()
	assert.NoError(err, "failed getting outputs")
	assert.NoError(outputs.Validate(), "failed validating outputs")
	assert.True(outputs.Count() == 1, "the transaction must contain one output")
	assert.True(outputs.ByRecipient(meHouse).Count() == 1, "the transaction must contain one output that names the recipient")
	house := &house.House{}
	assert.NoError(outputs.StateAt(0, house), "failed to get house state")
	assert.NotEmpty(house.LinearID, "the house must have a linear ID")
	assert.True(house.Valuation > 0, "the house must have a valuation")
	assert.NotEmpty(house.Address, "the house must have an address")

	// check action
	assert.Equal(otherCash, action.Recipient, "recipient mismatch")
	assert.True(action.Amount > 0, "amount must be greater than 0")
	assert.Equal("USD", action.Type, "currency must be USD")
	assert.Equal(meCash, action.From, "sender mismatch")

	// check house and action match
	assert.Equal(house.Valuation, action.Amount, "valuation mismatch")

	// Append the cash transfer to the transaction
	err = tx.Transfer(
		ttxcc.MyWalletFromTx(context, tx),
		action.Type,
		[]uint64{action.Amount},
		[]view.Identity{action.Recipient},
	)
	assert.NoError(err, "failed appending transfer")

	// Respond to the request to transfer the cash
	_, err = context.RunView(ttxcc.NewCollectActionsResponderView(tx, action))
	assert.NoError(err, "failed responding to action collect")

	// Sign and send back
	_, err = context.RunView(ttxcc.NewEndorseView(tx))
	assert.NoError(err, "failed to endorse transaction")

	// Wait for confirmation
	return context.RunView(ttxcc.NewFinalityView(tx))
}
