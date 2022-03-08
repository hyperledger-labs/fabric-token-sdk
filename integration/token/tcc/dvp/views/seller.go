/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package views

import (
	"encoding/json"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/token/tcc/dvp/views/house"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/nftcc"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxcc"
)

// Sell contains the sell instructions
type Sell struct {
	// Wallet is the wallet from which recipient identities must be derived
	Wallet string
	// HouseID is the house ID of the house to sell
	HouseID string
	// Buyer is the identity of the buyer (it is identifier as defined in the topology)
	Buyer string
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
	assert.NoError(err, "failed to create a new token transaction")

	// Prepare House Transfer
	tx, house, err := d.prepareHouseTransfer(context, tx)
	assert.NoError(err, "failed to prepare house transfer")

	// Prepare Payment
	tx, err = d.preparePayment(context, tx, house)
	assert.NoError(err, "failed to prepare payment")

	// Collect signature from the parties
	_, err = context.RunView(ttxcc.NewCollectEndorsementsView(tx))
	assert.NoError(err, "failed to collect endorsements")

	// Send to the ordering service and wait for confirmation
	return context.RunView(ttxcc.NewOrderingAndFinalityView(tx))
}

func (d *SellHouseView) preparePayment(context view.Context, tx *ttxcc.Transaction, house *house.House) (*ttxcc.Transaction, error) {
	// we need the house's valuation
	wallet := nftcc.MyWallet(context)
	assert.NotNil(wallet, "failed getting default wallet")

	// exchange pseudonyms for the token transfer
	me, other, err := ttxcc.ExchangeRecipientIdentities(context, d.Wallet, view.Identity(d.Buyer))
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

	return tx, nil
}

func (d *SellHouseView) prepareHouseTransfer(context view.Context, tx *ttxcc.Transaction) (*ttxcc.Transaction, *house.House, error) {
	// let's prepare the NFT transfer
	nfttx := nftcc.Wrap(tx)

	house := &house.House{}
	qe, err := nfttx.QueryExecutor()
	assert.NoError(err, "failed to create selector")
	assert.NoError(qe.QueryByKey(house, "LinearID", d.HouseID), "failed loading house with id %s", d.HouseID)

	buyer, err := nftcc.RequestRecipientIdentity(context, view.Identity(d.Buyer))
	assert.NoError(err, "failed getting buyer identity")

	wallet := nftcc.MyWallet(context)
	assert.NotNil(wallet, "failed getting default wallet")

	// Transfer ownership of the house to the buyer
	assert.NoError(nfttx.Transfer(wallet, house, buyer), "failed transferring house")

	return tx, house, nil
}

type SellHouseViewFactory struct{}

func (s SellHouseViewFactory) NewView(in []byte) (view.View, error) {
	f := &SellHouseView{Sell: &Sell{}}
	err := json.Unmarshal(in, f)
	assert.NoError(err, "failed unmarshalling input")
	return f, nil
}
