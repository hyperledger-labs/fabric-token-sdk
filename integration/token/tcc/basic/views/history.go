/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package views

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxcc"
)

type IssuerHistory struct {
	Wallet    string
	TokenType string
}

type IssuerHistoryView struct {
	*IssuerHistory
}

func (p *IssuerHistoryView) Call(context view.Context) (interface{}, error) {
	wallet := ttxcc.GetIssuerWallet(context, p.Wallet)
	return wallet.HistoryTokens(ttxcc.WithType(p.TokenType))
}

type IssuerHistoryViewFactory struct{}

func (i *IssuerHistoryViewFactory) NewView(in []byte) (view.View, error) {
	f := &IssuerHistoryView{IssuerHistory: &IssuerHistory{}}
	err := json.Unmarshal(in, f.IssuerHistory)
	assert.NoError(err, "failed unmarshalling input")
	return f, nil
}
