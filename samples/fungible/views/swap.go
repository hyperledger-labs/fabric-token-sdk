/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package views

import (
	"encoding/json"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

// Swap contains the input information for a swap
type Swap struct {
	// FromWallet is the wallet A will use
	FromWallet string
	// FromType is the token type A will transfer
	FromType string
	// FromQuantity is the amount A will transfer
	FromQuantity uint64
	// ToType is the token type To will transfer
	ToType string
	// ToQuantity is the amount To will transfer
	ToQuantity uint64
	// To is the identity of the To's FSC node
	To string
}

type SwapInitiatorView struct {
	*Swap
}

func (t *SwapInitiatorView) Call(context view.Context) (interface{}, error) {
	// As a first step operation, A contacts the recipient's FSC node
	// to exchange identities to use to assign ownership of the transferred tokens.
	me, other, err := ttx.ExchangeRecipientIdentities(context, t.FromWallet, view2.GetIdentityProvider(context).Identity(t.To))
	assert.NoError(err, "failed exchanging identities")

	// At this point, A is ready to prepare the token transaction.
	// A creates an anonymous transaction (this means that the resulting Fabric transaction will be signed using idemix, for example),
	// and specify the auditor that must be contacted to approve the operation.
	tx, err := ttx.NewAnonymousTransaction(
		context,
		ttx.WithAuditor(view2.GetIdentityProvider(context).Identity("auditor")),
	)
	assert.NoError(err, "failed creating transaction")

	// A will select tokens owned by this wallet
	senderWallet := ttx.GetWallet(context, t.FromWallet)
	assert.NotNil(senderWallet, "sender wallet [%s] not found", t.FromWallet)

	// A adds a new transfer operation to the transaction following the instruction received.
	err = tx.Transfer(
		senderWallet,
		t.FromType,
		[]uint64{t.FromQuantity},
		[]view.Identity{other},
	)
	assert.NoError(err, "failed adding output")

	// At this point, A is ready to collect To's transfer.
	// She does that by using the CollectActionsView.
	// A specifies the actions that she is expecting to be added to the transaction.
	// For each action, A contacts the recipient sending the transaction and the expected action.
	// At the end of the view, tx contains the collected actions
	_, err = context.RunView(ttx.NewCollectActionsView(tx,
		&ttx.ActionTransfer{
			From:      other,
			Type:      t.ToType,
			Amount:    t.ToQuantity,
			Recipient: me,
		},
	))
	assert.NoError(err, "failed collecting actions")

	// A doubles check that the content of the transaction is the one expected.
	assert.NoError(tx.IsValid(), "failed verifying transaction")

	outputs, err := tx.Outputs()
	assert.NoError(err, "failed getting outputs")
	os := outputs.ByRecipient(other)
	assert.Equal(0, os.Sum().Cmp(token2.NewQuantityFromUInt64(t.FromQuantity)))
	assert.Equal(os.Count(), os.ByType(t.FromType).Count())

	os = outputs.ByRecipient(me)
	assert.Equal(0, os.Sum().Cmp(token2.NewQuantityFromUInt64(t.ToQuantity)))
	assert.Equal(os.Count(), os.ByType(t.ToType).Count())

	// A is ready to collect all the required signatures and form the Transaction.
	_, err = context.RunView(ttx.NewCollectEndorsementsView(tx))
	assert.NoError(err, "failed to sign transaction")

	// Send to the ordering service and wait for finality
	_, err = context.RunView(ttx.NewOrderingAndFinalityView(tx))
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

type SwapResponderView struct{}

func (t *SwapResponderView) Call(context view.Context) (interface{}, error) {
	// As a first step, To responds to the request to exchange token recipient identities.
	// To takes his token recipient identity from the default wallet (ttx.MyWallet(context)),
	// if not otherwise specified.
	_, _, err := ttx.RespondExchangeRecipientIdentities(context)
	assert.NoError(err, "failed getting identity")

	// To respond to a call from the CollectActionsView, the first thing to do is to receive
	// the transaction and the requested action.
	// This could happen multiple times, depending on the use-case.
	tx, action, err := ttx.ReceiveAction(context)
	assert.NoError(err, "failed receiving action")

	// Depending on the use case, To can further analyse the requested action, before proceeding. It depends on the use-case.
	// If everything is fine, To adds his transfer to A as requested.
	// To will select tokens from his default wallet matching the transaction
	bobWallet := ttx.MyWalletFromTx(context, tx)
	assert.NotNil(bobWallet, "To's default wallet not found")
	err = tx.Transfer(
		bobWallet,
		action.Type,
		[]uint64{action.Amount},
		[]view.Identity{action.Recipient},
	)
	assert.NoError(err, "failed appending transfer")

	// Once To finishes the preparation of his part, he can send Back the transaction
	// calling the CollectActionsResponderView
	_, err = context.RunView(ttx.NewCollectActionsResponderView(tx, action))
	assert.NoError(err, "failed responding to action collect")

	// If everything is fine, To endorses and sends back his signature.
	_, err = context.RunView(ttx.NewEndorseView(tx))
	assert.NoError(err, "failed endorsing transaction")

	// Before completing, the recipient waits for finality of the transaction
	_, err = context.RunView(ttx.NewFinalityView(tx))
	assert.NoError(err, "new tokens were not committed")

	return tx.ID(), nil
}
