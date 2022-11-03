/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package htlc

import (
	"encoding/json"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/htlc"
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
	senderWallet := htlc.GetWallet(context, r.Wallet, token.WithTMSID(r.TMSID))
	assert.NotNil(senderWallet, "sender wallet [%s] not found", r.Wallet)

	expired, err := htlc.Wallet(context, senderWallet).ListExpired()
	assert.NoError(err, "cannot retrieve list of expired tokens")
	assert.True(expired.Count() > 0, "no htlc script has expired")

	tx, err := htlc.NewAnonymousTransaction(
		context,
		ttx.WithAuditor(view2.GetIdentityProvider(context).Identity("auditor")),
		ttx.WithTMSID(r.TMSID),
	)
	assert.NoError(err, "failed to create an htlc transaction")
	for _, id := range expired.Tokens {
		assert.NoError(tx.Reclaim(senderWallet, id), "failed adding a reclaim for [%s]", id)
	}

	// The sender is ready to collect all the required signatures.
	// In this case, the sender's and the auditor's signatures.
	// Invoke the Token Chaincode to collect endorsements on the Token Request and prepare the relative transaction.
	// This is all done in one shot running the following view.
	_, err = context.RunView(htlc.NewCollectEndorsementsView(tx))
	assert.NoError(err, "failed to collect endorsements on htlc transaction")

	// Last but not least, the transaction is sent for ordering, and we wait for transaction finality.
	_, err = context.RunView(htlc.NewOrderingAndFinalityView(tx))
	assert.NoError(err, "failed to commit htlc transaction")

	return tx.ID(), nil
}

type ReclaimAllViewFactory struct{}

func (p *ReclaimAllViewFactory) NewView(in []byte) (view.View, error) {
	f := &ReclaimAllView{ReclaimAll: &ReclaimAll{}}
	err := json.Unmarshal(in, f.ReclaimAll)
	assert.NoError(err, "failed unmarshalling input")

	return f, nil
}
