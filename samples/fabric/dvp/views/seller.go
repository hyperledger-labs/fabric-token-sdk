/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package views

import (
	"encoding/json"
	"github.com/hyperledger-labs/fabric-token-sdk/samples/fabric/dvp/views/house"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/nftcc"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxcc"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
)

type Sell struct {
	Wallet  string
	HouseID string
	Buyer   view.Identity
}

type SellHouseView struct {
	*Sell
}

func (d *SellHouseView) Call(context view.Context) (interface{}, error) {
	// Prepare a new token transaction.
	// It will contain two legs:
	// 1. The first leg will be used to transfer the house to the buyer.
	// 2. The second leg will be used to transfer the cash to the seller.
	tx, err := ttxcc.NewAnonymousTransaction(
		context,
		ttxcc.WithAuditor(
			fabric.GetDefaultIdentityProvider(context).Identity("auditor"), // Retrieve the auditor's FSC node identity
		),
	)
	assert.NoError(err)

	// Prepare Payment
	tx, house, err := d.preparePayment(context, tx)
	assert.NoError(err, "failed to prepare payment")

	// Prepare House Transfer
	tx, err = d.prepareHouseTransfer(context, tx, house)
	assert.NoError(err, "failed to prepare house transfer")

	// Collect signature from the parties
	_, err = context.RunView(ttxcc.NewCollectEndorsementsView(tx))
	assert.NoError(err, "failed to collect endorsements")

	// Send to the ordering service and wait for confirmation
	return context.RunView(ttxcc.NewOrderingAndFinalityView(tx))
}

func (d *SellHouseView) preparePayment(context view.Context, tx *ttxcc.Transaction) (*ttxcc.Transaction, *house.House, error) {
	// we need house's valuation, let's load the state from the world state
	house := &house.House{}
	assert.NoError(
		nftcc.GetVault(context).GetState("house", d.HouseID, house),
		"failed loading house with id %s", d.HouseID,
	)

	// exchange pseudonyms for the token transfer
	me, other, err := ttxcc.ExchangeRecipientIdentities(context, d.Wallet, d.Buyer)
	assert.NoError(err, "failed exchanging identities")

	// collect token transfer from the buyer
	_, err = context.RunView(ttxcc.NewCollectActionsView(tx,
		&ttxcc.ActionTransfer{
			From:      other,
			Type:      "USD",
			Amount:    house.Valuation,
			Recipient: me,
		}))
	assert.NoError(err, "failed collecting token action")

	return tx, house, nil
}

func (d *SellHouseView) prepareHouseTransfer(context view.Context, tx *ttxcc.Transaction, house *house.House) (*ttxcc.Transaction, error) {
	// let's prepare the NFT transfer
	nfttx := nftcc.Wrap(tx)

	buyer, err := nftcc.RequestRecipientIdentity(context, d.Buyer)
	assert.NoError(err, "failed getting buyer identity")

	// Transfer ownership of the house to the buyer
	assert.NoError(nfttx.Transfer(house, buyer), "failed transferring house")

	return tx, nil
}

type SellHouseViewFactory struct{}

func (s SellHouseViewFactory) NewView(in []byte) (view.View, error) {
	f := &SellHouseView{Sell: &Sell{}}
	err := json.Unmarshal(in, f)
	assert.NoError(err, "failed unmarshalling input")
	return f, nil
}
