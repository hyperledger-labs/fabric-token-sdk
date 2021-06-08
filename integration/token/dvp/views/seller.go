/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package views

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/services/endorser"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/services/state"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
)

type Sell struct {
	Wallet  string
	HouseID string
	Buyer   view.Identity

	Approvers []view.Identity
}

type SellHouseView struct {
	*Sell
}

func (d *SellHouseView) Call(context view.Context) (interface{}, error) {
	// prepare a new fabric transaction
	_, tx, err := endorser.NewTransaction(context)
	assert.NoError(err)
	tx.SetProposal("house", "Version-0.0", "sell")

	// Prepare Payment
	tx, err = d.preparePayment(tx, context)
	assert.NoError(err)

	// Prepare House Transfer
	tx, err = d.prepareHouseTransfer(tx)
	assert.NoError(err)

	// Collect signature from the parties
	_, err = context.RunView(endorser.NewCollectEndorsementsView(tx, context.Me(), d.Buyer))
	assert.NoError(err)

	// Collect signature from zkat auditor signature
	zkatTx, err := ttx.Wrap(tx, ttx.WithAuditor(fabric.GetIdentityProvider(context).Identity("auditor")))
	assert.NoError(err)
	_, err = context.RunView(ttx.NewCollectAuditorEndorsement(zkatTx))
	assert.NoError(err)

	// Collect signatures from the approvers but without sending metadata
	_, err = context.RunView(endorser.NewCollectApprovesView(tx, d.Approvers...))
	assert.NoError(err)

	// Send to the ordering service and wait for confirmation
	return context.RunView(endorser.NewOrderingView(tx))
}

func (d *SellHouseView) preparePayment(tx *endorser.Transaction, context view.Context) (*endorser.Transaction, error) {
	// we need house's valuation, let's load the state from the world state
	house := &House{}
	assert.NoError(state.GetWorldState(context).GetState("house", d.HouseID, house), "failed loading house with id %s", d.HouseID)

	// exchange pseudonyms for the token transfer
	me, other, err := ttx.ExchangeRecipientIdentitiesInitiator(context, d.Wallet, d.Buyer)
	assert.NoError(err, "failed exchanging identities")

	// collect token transfer from the buyer
	tokenTx, err := ttx.Wrap(tx)
	assert.NoError(err)
	_, err = context.RunView(ttx.NewCollectActionsView(tokenTx,
		&ttx.ActionTransfer{
			From:      other,
			Type:      "USD",
			Amount:    house.Valuation,
			Recipient: me,
		}))
	assert.NoError(err, "failed collecting token action")

	return tx, nil
}

func (d *SellHouseView) prepareHouseTransfer(tx *endorser.Transaction) (*endorser.Transaction, error) {
	// let's use the state package to hide the complexity of the rws management
	// with a state-oriented programming
	stx, err := state.Wrap(tx)
	assert.NoError(err)

	// Add dependency to the existing state
	house := &House{}
	assert.NoError(stx.AddInputByLinearID(d.HouseID, house))
	// Update the owner field
	house.Owner = d.Buyer
	assert.NoError(stx.AddOutput(house))

	return tx, nil
}

type SellHouseViewFactory struct{}

func (s SellHouseViewFactory) NewView(in []byte) (view.View, error) {
	f := &SellHouseView{Sell: &Sell{}}
	err := json.Unmarshal(in, f)
	assert.NoError(err, "failed unmarshalling input")
	return f, nil
}
