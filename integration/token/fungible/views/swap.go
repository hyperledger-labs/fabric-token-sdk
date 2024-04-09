/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package views

import (
	"encoding/json"
	"math/big"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
)

// Swap contains the input information for a swap
type Swap struct {
	// Auditor is the identity of the auditor that must be contacted to approve the transaction
	Auditor string
	// AliceWallet is the wallet Alice will use
	AliceWallet string
	// FromAliceType is the token type Alice will transfer
	FromAliceType string
	// FromAliceAmount is the amount Alice will transfer
	FromAliceAmount uint64
	// FromBobType is the token type Bob will transfer
	FromBobType string
	// FromBobAmount is the amount Bob will transfer
	FromBobAmount uint64
	// Bob is the identity of the Bob's FSC node
	Bob view.Identity
}

type SwapInitiatorView struct {
	*Swap
}

func (t *SwapInitiatorView) Call(context view.Context) (interface{}, error) {
	// As a first step operation, Alice contacts the recipient's FSC node
	// to exchange identities to use to assign ownership of the transferred tokens.
	me, other, err := ttx.ExchangeRecipientIdentities(context, t.AliceWallet, t.Bob)
	assert.NoError(err, "failed exchanging identities")

	// At this point, Alice is ready to prepare the token transaction.
	// Alice creates an anonymous transaction (this means that the resulting Fabric transaction will be signed using idemix, for example),
	// and specify the auditor that must be contacted to approve the operation.
	tx, err := ttx.NewAnonymousTransaction(
		context,
		ttx.WithAuditor(view2.GetIdentityProvider(context).Identity(t.Auditor)),
	)
	assert.NoError(err, "failed creating transaction")

	// Alice will select tokens owned by this wallet
	senderWallet := ttx.GetWallet(context, t.AliceWallet)
	assert.NotNil(senderWallet, "sender wallet [%s] not found", t.AliceWallet)

	// Alice adds a new transfer operation to the transaction following the instruction received.
	err = tx.Transfer(
		senderWallet,
		t.FromAliceType,
		[]uint64{t.FromAliceAmount},
		[]view.Identity{other},
	)
	assert.NoError(err, "failed adding output")

	// At this point, Alice is ready to collect To's transfer.
	// She does that by using the CollectActionsView.
	// Alice specifies the actions that she is expecting to be added to the transaction.
	// For each action, Alice contacts the recipient sending the transaction and the expected action.
	// At the end of the view, tx contains the collected actions
	_, err = context.RunView(ttx.NewCollectActionsView(tx,
		&ttx.ActionTransfer{
			From:      other,
			Type:      t.FromBobType,
			Amount:    t.FromBobAmount,
			Recipient: me,
		},
	))
	assert.NoError(err, "failed collecting actions")

	// Alice doubles check that the content of the transaction is the one expected.
	assert.NoError(tx.IsValid(), "failed verifying transaction")

	outputs, err := tx.Outputs()
	assert.NoError(err, "failed getting outputs")
	os := outputs.ByRecipient(other)
	assert.Equal(0, os.Sum().Cmp(big.NewInt(int64(t.FromAliceAmount))))
	assert.Equal(os.Count(), os.ByType(t.FromAliceType).Count())

	os = outputs.ByRecipient(me)
	assert.Equal(0, os.Sum().Cmp(big.NewInt(int64(t.FromBobAmount))))
	assert.Equal(os.Count(), os.ByType(t.FromBobType).Count())

	// Alice is ready to collect all the required signatures and form the Transaction.
	_, err = context.RunView(ttx.NewCollectEndorsementsView(tx))
	assert.NoError(err, "failed to sign transaction")

	// Sanity checks:
	// - the transaction is in busy state in the vault
	net := network.GetInstance(context, tx.Network(), tx.Channel())
	vault, err := net.Vault(tx.Namespace())
	assert.NoError(err, "failed to retrieve vault [%s]", tx.Namespace())
	vc, _, err := vault.Status(tx.ID())
	assert.NoError(err, "failed to retrieve vault status for transaction [%s]", tx.ID())
	assert.Equal(network.Busy, vc, "transaction [%s] should be in busy state", tx.ID())

	// Send to the ordering service and wait for finality
	_, err = context.RunView(ttx.NewOrderingAndFinalityView(tx))
	assert.NoError(err, "failed asking ordering")

	// Sanity checks:
	// - the transaction is in valid state in the vault
	vc, _, err = vault.Status(tx.ID())
	assert.NoError(err, "failed to retrieve vault status for transaction [%s]", tx.ID())
	assert.Equal(network.Valid, vc, "transaction [%s] should be in valid state", tx.ID())

	// Check that the tokens are or are not in the vault
	AssertTokensInVault(vault, tx, outputs, me)

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
	me, _, err := ttx.RespondExchangeRecipientIdentities(context)
	assert.NoError(err, "failed getting identity")

	// To respond to a call from the CollectActionsView, the first thing to do is to receive
	// the transaction and the requested action.
	// This could happen multiple times, depending on the use-case.
	tx, action, err := ttx.ReceiveAction(context)
	assert.NoError(err, "failed receiving action")

	// Depending on the use case, To can further analyse the requested action, before proceeding. It depends on the use-case.
	// If everything is fine, To adds his transfer to Alice as requested.
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

	// Sanity checks:
	// - the transaction is in busy state in the vault
	net := network.GetInstance(context, tx.Network(), tx.Channel())
	vault, err := net.Vault(tx.Namespace())
	assert.NoError(err, "failed to retrieve vault [%s]", tx.Namespace())
	vc, _, err := vault.Status(tx.ID())
	assert.NoError(err, "failed to retrieve vault status for transaction [%s]", tx.ID())
	assert.Equal(network.Busy, vc, "transaction [%s] should be in busy state", tx.ID())

	// Before completing, the recipient waits for finality of the transaction
	_, err = context.RunView(ttx.NewFinalityView(tx))
	assert.NoError(err, "new tokens were not committed")

	vc, _, err = vault.Status(tx.ID())
	assert.NoError(err, "failed to retrieve vault status for transaction [%s]", tx.ID())
	assert.Equal(network.Valid, vc, "transaction [%s] should be in valid state", tx.ID())

	// Check that the tokens are or are not in the vault
	outputs, err := tx.Outputs()
	assert.NoError(err, "failed to retrieve outputs")
	AssertTokensInVault(vault, tx, outputs, me)

	return tx.ID(), nil
}
