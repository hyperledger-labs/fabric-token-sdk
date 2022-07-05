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

type Claim struct {
	TMSID    token.TMSID
	Wallet   string
	PreImage []byte // PreImage of the exchange script to spend
}

type ClaimView struct {
	*Claim
}

func (r *ClaimView) Call(context view.Context) (interface{}, error) {
	claimWallet := ttx.GetWallet(context, r.Wallet, token.WithTMSID(r.TMSID))
	assert.NotNil(claimWallet, "wallet [%s] not found", r.Wallet)

	matched, err := exchange.Wallet(context, claimWallet, token.WithTMSID(r.TMSID)).ListByPreImage(r.PreImage)
	assert.NoError(err, "cannot retrieve list of expired exchange")
	assert.True(len(matched) == 1, "expected only one exchange script to match, got [%d]", len(matched))

	tx, err := exchange.NewTransaction(
		context,
		fabric.GetIdentityProvider(context, r.TMSID.Network).DefaultIdentity(),
		ttx.WithAuditor(view2.GetIdentityProvider(context).Identity("auditor")),
		ttx.WithTMSID(r.TMSID),
	)
	assert.NoError(err, "failed created an exchange transaction")
	assert.NoError(tx.Claim(claimWallet, matched[0], r.PreImage), "failed adding a claim for [%s]", matched[0].Id)

	_, err = context.RunView(ttx.NewCollectEndorsementsView(tx.Transaction))
	assert.NoError(err, "failed to collect endorsements on exchange transaction")

	_, err = context.RunView(ttx.NewOrderingAndFinalityView(tx.Transaction))
	assert.NoError(err, "failed to commit issue transaction")

	return tx.ID(), nil
}

type ClaimViewFactory struct{}

func (p *ClaimViewFactory) NewView(in []byte) (view.View, error) {
	f := &ClaimView{Claim: &Claim{}}
	err := json.Unmarshal(in, f.Claim)
	assert.NoError(err, "failed unmarshalling input")

	return f, nil
}
