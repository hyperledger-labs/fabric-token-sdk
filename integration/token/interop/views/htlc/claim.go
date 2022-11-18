/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package htlc

import (
	"encoding/json"
	"fmt"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/htlc"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/pkg/errors"
)

// Claim contains the input information to claim a token
type Claim struct {
	// TMSID identifies the TMS to use to perform the token operation
	TMSID token.TMSID
	// Wallet is the identifier of the wallet to use
	Wallet string
	// PreImage of the hash encoded in the htlc script in the token to be claimed
	PreImage []byte
}

type ClaimView struct {
	*Claim
}

func (r *ClaimView) Call(context view.Context) (res interface{}, err error) {
	var tx *htlc.Transaction
	defer func() {
		if e := recover(); e != nil {
			if tx != nil {
				fmt.Printf("add to err tx id [%s]", tx.ID())
				err = errors.Errorf("<<<[%s]>>>: %s", tx.ID(), err)
			}
		}
	}()

	claimWallet := htlc.GetWallet(context, r.Wallet, token.WithTMSID(r.TMSID))
	assert.NotNil(claimWallet, "wallet [%s] not found", r.Wallet)

	matched, err := htlc.Wallet(context, claimWallet).ListByPreImage(r.PreImage)
	assert.NoError(err, "htlc script has expired")
	assert.True(matched.Count() == 1, "expected only one htlc script to match, got [%d]", matched.Count())

	tx, err = htlc.NewAnonymousTransaction(
		context,
		ttx.WithAuditor(view2.GetIdentityProvider(context).Identity("auditor")),
		ttx.WithTMSID(r.TMSID),
	)
	assert.NoError(err, "failed to create an htlc transaction")
	assert.NoError(tx.Claim(claimWallet, matched.At(0), r.PreImage), "failed adding a claim for [%s]", matched.At(0).Id)

	_, err = context.RunView(htlc.NewCollectEndorsementsView(tx))
	assert.NoError(err, "failed to collect endorsements on htlc transaction")

	_, err = context.RunView(htlc.NewOrderingAndFinalityView(tx))
	assert.NoError(err, "failed to commit htlc transaction")

	return tx.ID(), nil
}

type ClaimViewFactory struct{}

func (p *ClaimViewFactory) NewView(in []byte) (view.View, error) {
	f := &ClaimView{Claim: &Claim{}}
	err := json.Unmarshal(in, f.Claim)
	assert.NoError(err, "failed unmarshalling input")

	return f, nil
}
