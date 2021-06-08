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
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
)

type IssueCash struct {
	Wallet   string
	Receiver view.Identity
	Typ      string
	Quantity uint64

	Approver view.Identity
}

type IssueCashView struct {
	*IssueCash
}

func (p *IssueCashView) Call(context view.Context) (interface{}, error) {
	recipient, err := ttx.RequestRecipientIdentity(context, p.Receiver)
	assert.NoError(err, "failed getting recipient identity")

	// Prepare transaction
	tx, err := ttx.NewTransaction(context, ttx.WithAuditor(fabric.GetIdentityProvider(context).Identity("auditor")))
	assert.NoError(err)

	wallet := ttx.GetIssuerWallet(context, p.Wallet)
	assert.NoError(tx.Issue(wallet, recipient, p.Typ, p.Quantity), "failed issuing token")

	_, err = context.RunView(ttx.NewCollectEndorsementsView(tx))
	assert.NoError(err, "failed collecting endorsement")

	_, err = context.RunView(ttx.NewCollectApprovesView(tx, p.Approver))
	assert.NoError(err, "failed collecting endorsement")

	// Send to the ordering service and wait for confirmation
	_, err = context.RunView(ttx.NewOrderingView(tx))
	assert.NoError(err, "failed asking ordering")
	return tx.ID(), nil
}

type IssueCashViewFactory struct {
}

func (p *IssueCashViewFactory) NewView(in []byte) (view.View, error) {
	f := &IssueCashView{IssueCash: &IssueCash{}}
	err := json.Unmarshal(in, f.IssueCash)
	assert.NoError(err, "failed unmarshalling input")
	return f, nil
}
