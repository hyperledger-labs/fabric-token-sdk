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
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/nftcc"
)

// Transfer contains the transfer instructions
type Transfer struct {
	// Wallet is the wallet from which recipient identities must be derived
	Wallet string
	// HouseID is the house ID of the house to sell
	HouseID string
	// Recipient is the identity of the buyer (it is identifier as defined in the topology)
	Recipient string
}

type TransferHouseView struct {
	*Transfer
}

func (d *TransferHouseView) Call(context view.Context) (interface{}, error) {
	// Prepare a new token transaction.
	tx, err := nftcc.NewAnonymousTransaction(
		context,
		nftcc.WithAuditor(
			fabric.GetDefaultIdentityProvider(context).Identity("auditor"), // Retrieve the auditor's FSC node identity
		),
	)
	assert.NoError(err, "failed to create a new token transaction")

	buyer, err := nftcc.RequestRecipientIdentity(context, view.Identity(d.Recipient))
	assert.NoError(err, "failed getting buyer identity")

	wallet := nftcc.MyWallet(context)
	assert.NotNil(wallet, "failed getting default wallet")

	// Transfer ownership of the house to the buyer
	house := &House{}
	assert.NoError(nftcc.MyWallet(context).QueryByKey(house, "LinearID", d.HouseID), "failed loading house with id %s", d.HouseID)

	assert.NoError(tx.Transfer(wallet, house, buyer), "failed transferring house")

	// Collect signature from the parties
	_, err = context.RunView(nftcc.NewCollectEndorsementsView(tx))
	assert.NoError(err, "failed to collect endorsements")

	// Send to the ordering service and wait for confirmation
	_, err = context.RunView(nftcc.NewOrderingAndFinalityView(tx))
	assert.NoError(err, "failed to order and finalize")

	return tx.ID(), nil
}

type TransferHouseViewFactory struct{}

func (s TransferHouseViewFactory) NewView(in []byte) (view.View, error) {
	f := &TransferHouseView{Transfer: &Transfer{}}
	err := json.Unmarshal(in, f)
	assert.NoError(err, "failed unmarshalling input")
	return f, nil
}
