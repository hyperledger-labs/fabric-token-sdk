/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package exchange

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/exchange"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
)

// ReclaimAll contains the input information to reclaim tokens
type ReclaimAll struct {
	// TMSID identifies the TMS to use to perform the token operation
	TMSID token.TMSID
	// Wallet is the identifier of the wallet that owns the tokens to reclaim
	Wallet string
}

// ReclaimAllView is the view of a party who wants to reclaim all previously locked tokens with an expired timeout
type ReclaimAllView struct {
	*ReclaimAll
}

func (r *ReclaimAllView) Call(context view.Context) (interface{}, error) {
	// The sender will select tokens owned by this wallet
	senderWallet := exchange.GetWallet(context, r.Wallet, token.WithTMSID(r.TMSID))
	assert.NotNil(senderWallet, "sender wallet [%s] not found", r.Wallet)

	expired, err := exchange.Wallet(context, senderWallet, token.WithTMSID(r.TMSID)).ListExpired()
	assert.NoError(err, "cannot retrieve list of expired exchanges")
	assert.True(len(expired) > 0, "no exchange script has expired")

	tx, err := exchange.NewTransaction(
		context,
		fabric.GetDefaultIdentityProvider(context).DefaultIdentity(),
		ttx.WithAuditor(view2.GetIdentityProvider(context).Identity("auditor")),
		ttx.WithTMSID(r.TMSID),
	)
	assert.NoError(err, "failed to create an exchange transaction")
	for _, id := range expired {
		assert.NoError(tx.Reclaim(senderWallet, id), "failed adding a reclaim for [%s]", id)
	}

	// The sender is ready to collect all the required signatures.
	// In this case, the sender's and the auditor's signatures.
	// Invoke the Token Chaincode to collect endorsements on the Token Request and prepare the relative transaction.
	// This is all done in one shot running the following view.
	_, err = context.RunView(exchange.NewCollectEndorsementsView(tx))
	assert.NoError(err, "failed to collect endorsements on exchange transaction")

	// Last but not least, the transaction is sent for ordering, and we wait for transaction finality.
	_, err = context.RunView(exchange.NewOrderingAndFinalityView(tx))
	assert.NoError(err, "failed to commit exchange transaction")

	return tx.ID(), nil
}

type ReclaimAllViewFactory struct{}

func (p *ReclaimAllViewFactory) NewView(in []byte) (view.View, error) {
	f := &ReclaimAllView{ReclaimAll: &ReclaimAll{}}
	err := json.Unmarshal(in, f.ReclaimAll)
	assert.NoError(err, "failed unmarshalling input")

	return f, nil
}
