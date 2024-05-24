/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pledge

import (
	"encoding/json"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/pledge"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

// Redeem contains the input information for a redeem
type Redeem struct {
	TMSID token2.TMSID
	// TokenID contains the identifier of the token to be redeemed
	TokenID *token.ID
}

type RedeemView struct {
	*Redeem
}

func (rv *RedeemView) Call(context view.Context) (interface{}, error) {
	w, err := pledge.GetIssuerWallet(context, "")
	assert.NoError(err, "failed to retrieve wallet of issuer during redeem")

	wallet := pledge.NewIssuerWallet(context, w)
	t, script, err := wallet.GetPledgedToken(rv.TokenID)
	assert.NoError(err, "failed to retrieve pledged token during redeem")

	// make sure token was in fact claimed in the other network
	proof, err := pledge.RequestProofOfTokenWithMetadataExistence(context, rv.TokenID, rv.TMSID, script)
	assert.NoError(err, "failed to retrieve and verify proof of token existence")

	// Create a new transaction
	tx, err := pledge.NewAnonymousTransaction(
		context,
		ttx.WithAuditor(view2.GetIdentityProvider(context).Identity("auditor")),
		ttx.WithTMSID(rv.TMSID),
	)
	assert.NoError(err, "failed created a new transaction")

	ow, err := pledge.GetOwnerWallet(context, "")
	assert.NoError(err, "failed to retrieve owner wallet")

	err = tx.RedeemPledge(ow, t, rv.TokenID, proof)
	assert.NoError(err, "failed adding redeem")

	// Collect signatures
	_, err = context.RunView(pledge.NewCollectEndorsementsView(tx))
	assert.NoError(err, "failed to sign redeem transaction")

	// Sends the transaction for ordering and wait for transaction finality
	_, err = context.RunView(pledge.NewOrderingAndFinalityView(tx))
	assert.NoError(err, "failed to commit redeem transaction")

	return tx.ID(), nil
}

type RedeemViewFactory struct{}

func (rvf *RedeemViewFactory) NewView(in []byte) (view.View, error) {
	v := &RedeemView{Redeem: &Redeem{}}
	err := json.Unmarshal(in, v.Redeem)
	assert.NoError(err, "failed unmarshalling input")

	return v, nil
}
