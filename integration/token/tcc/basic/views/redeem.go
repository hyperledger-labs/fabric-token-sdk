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

	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxcc"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type Redeem struct {
	Wallet   string
	TokenIDs []*token.Id
	Type     string
	Amount   uint64
}

type RedeemView struct {
	*Redeem
}

func (t *RedeemView) Call(context view.Context) (interface{}, error) {
	// Prepare transaction
	tx, err := ttxcc.NewAnonymousTransaction(
		context,
		ttxcc.WithAuditor(fabric.GetIdentityProvider(context).Identity("auditor")),
	)
	assert.NoError(err, "failed creating transaction")

	err = tx.Redeem(
		ttxcc.GetWallet(context, t.Wallet),
		t.Type,
		t.Amount,
		token2.WithTokenIDs(t.TokenIDs...),
	)
	assert.NoError(err, "failed adding new tokens")

	_, err = context.RunView(ttxcc.NewCollectEndorsementsView(tx))
	assert.NoError(err, "failed to sign transaction")

	// Send to the ordering service and wait for confirmation
	_, err = context.RunView(ttxcc.NewOrderingView(tx))
	assert.NoError(err, "failed asking ordering")

	return tx.ID(), nil
}

type RedeemViewFactory struct{}

func (p *RedeemViewFactory) NewView(in []byte) (view.View, error) {
	f := &RedeemView{Redeem: &Redeem{}}
	err := json.Unmarshal(in, f.Redeem)
	assert.NoError(err, "failed unmarshalling input")
	return f, nil
}
