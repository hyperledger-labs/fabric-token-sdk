/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package views

import (
	"encoding/json"

	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxcc"
)

type Swap struct {
	Wallet      string
	TypeLeft    string
	AmountLeft  uint64
	TypeRight   string
	AmountRight uint64
	Recipient   view.Identity
}

type SwapInitiatorView struct {
	*Swap
}

func (t *SwapInitiatorView) Call(context view.Context) (interface{}, error) {
	me, other, err := ttxcc.ExchangeRecipientIdentitiesInitiator(context, t.Wallet, t.Recipient)
	assert.NoError(err, "failed exchanging identities")

	// Prepare transaction
	tx, err := ttxcc.NewAnonymousTransaction(
		context,
		ttxcc.WithAuditor(fabric.GetIdentityProvider(context).Identity("auditor")),
	)
	assert.NoError(err, "failed creating transaction")

	err = tx.Transfer(ttxcc.GetWallet(context, t.Wallet), t.TypeLeft, []uint64{t.AmountLeft}, []view.Identity{other})
	assert.NoError(err, "failed adding output")

	_, err = context.RunView(ttxcc.NewCollectActionsView(tx,
		&ttxcc.ActionTransfer{
			From:      other,
			Type:      t.TypeRight,
			Amount:    t.AmountRight,
			Recipient: me,
		},
	))
	assert.NoError(err, "failed collecting actions")

	// check the content of the transaction
	assert.NoError(tx.Verify(), "failed verifying transaction")

	outputs, err := tx.Outputs()
	assert.NoError(err, "failed getting outputs")
	os := outputs.ByRecipient(other)
	assert.Equal(0, os.Sum().Cmp(token2.NewQuantityFromUInt64(uint64(t.AmountLeft))))
	assert.Equal(os.Count(), os.ByType(t.TypeLeft).Count())

	os = outputs.ByRecipient(me)
	assert.Equal(0, os.Sum().Cmp(token2.NewQuantityFromUInt64(uint64(t.AmountRight))))
	assert.Equal(os.Count(), os.ByType(t.TypeRight).Count())

	_, err = context.RunView(ttxcc.NewCollectEndorsementsView(tx))
	assert.NoError(err, "failed collecting endorsement")

	// Send to the ordering service and wait for confirmation
	_, err = context.RunView(ttxcc.NewOrderingView(tx))
	assert.NoError(err, "failed asking ordering")

	return tx.ID(), nil
}

type SwapInitiatorViewFactory struct{}

func (p *SwapInitiatorViewFactory) NewView(in []byte) (view.View, error) {
	f := &SwapInitiatorView{Swap: &Swap{}}
	err := json.Unmarshal(in, f.Swap)
	assert.NoError(err, "failed unmarshalling input")
	return f, nil
}

type SwapResponderView struct {
}

func (t *SwapResponderView) Call(context view.Context) (interface{}, error) {
	_, _, err := ttxcc.ExchangeRecipientIdentitiesResponder(context)
	assert.NoError(err, "failed getting identity")

	tx, action, err := ttxcc.ReceiveAction(context)
	assert.NoError(err, "failed receiving action")

	err = tx.Transfer(
		ttxcc.MyWalletForChannel(context, tx.Channel()),
		action.Type, []uint64{action.Amount}, []view.Identity{action.Recipient},
	)
	assert.NoError(err, "failed appending transfer")

	_, err = context.RunView(ttxcc.NewCollectActionsResponderView(tx, action))
	assert.NoError(err, "failed responding to action collect")

	// Endorse and send back
	_, err = context.RunView(ttxcc.NewEndorseView(tx))
	assert.NoError(err, "failed endorsing transaction")

	// Wait for finality
	_, err = context.RunView(ttxcc.NewFinalityView(tx))
	assert.NoError(err)

	return tx.ID(), nil
}
