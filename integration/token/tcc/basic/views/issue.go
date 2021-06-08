/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package views

import (
	"encoding/json"
	"fmt"

	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxcc"
)

type RegisterIssuer struct {
	TokenTypes []string
}

type RegisterIssuerView struct {
	*RegisterIssuer
}

func (r *RegisterIssuerView) Call(context view.Context) (interface{}, error) {
	for _, tokenType := range r.TokenTypes {
		_, err := context.RunView(ttxcc.NewRegisterIssuerIdentityView(tokenType))
		assert.NoError(err, "failed registering issuer identity for token type [%s]", tokenType)
	}

	return nil, nil
}

type RegisterIssuerViewFactory struct{}

func (p *RegisterIssuerViewFactory) NewView(in []byte) (view.View, error) {
	f := &RegisterIssuerView{RegisterIssuer: &RegisterIssuer{}}
	err := json.Unmarshal(in, f.RegisterIssuer)
	assert.NoError(err, "failed unmarshalling input")

	return f, nil
}

type IssueCash struct {
	Wallet    string
	TokenType string
	Quantity  uint64
	Recipient view.Identity
}

type IssueCashView struct {
	*IssueCash
}

func (p *IssueCashView) Call(context view.Context) (interface{}, error) {
	recipient, err := ttxcc.RequestRecipientIdentity(context, p.Recipient)
	assert.NoError(err, "failed getting recipient identity")

	// Limits
	wallet := ttxcc.GetIssuerWallet(context, p.Wallet)

	if p.TokenType == "USD" {
		// check limit
		history, err := wallet.HistoryTokens(ttxcc.WithType(p.TokenType))
		assert.NoError(err, "failed getting history for token type [%s]", p.TokenType)
		fmt.Printf("History [%s,%s]<[220]?\n", history.Sum(64).ToBigInt().Text(10), p.TokenType)
		assert.True(history.Sum(64).Add(token2.NewQuantityFromUInt64(p.Quantity)).Cmp(token2.NewQuantityFromUInt64(230)) <= 0)
	}

	// Prepare transaction
	tx, err := ttxcc.NewTransaction(
		context,
		fabric.GetIdentityProvider(context).DefaultIdentity(),
		ttxcc.WithAuditor(fabric.GetIdentityProvider(context).Identity("auditor")),
	)
	assert.NoError(err, "failed creating issue transaction")

	err = tx.Issue(
		wallet,
		recipient,
		p.TokenType,
		p.Quantity,
	)
	assert.NoError(err, "failed adding new issued token")

	_, err = context.RunView(ttxcc.NewCollectEndorsementsView(tx))
	assert.NoError(err, "failed to sign issue transaction")

	// Send to the ordering service and wait for confirmation
	_, err = context.RunView(ttxcc.NewOrderingView(tx))
	assert.NoError(err, "failed to commit issue transaction")

	return tx.ID(), nil
}

type IssueCashViewFactory struct{}

func (p *IssueCashViewFactory) NewView(in []byte) (view.View, error) {
	f := &IssueCashView{IssueCash: &IssueCash{}}
	err := json.Unmarshal(in, f.IssueCash)
	assert.NoError(err, "failed unmarshalling input")

	return f, nil
}
