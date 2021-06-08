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

type ListUnspentTokens struct {
	Wallet    string
	TokenType string
}

type ListUnspentTokensView struct {
	*ListUnspentTokens
}

func (p *ListUnspentTokensView) Call(context view.Context) (interface{}, error) {
	wallet := ttxcc.GetWallet(context, p.Wallet)
	return wallet.ListTokens(ttxcc.WithType(p.TokenType))
}

type ListUnspentTokensViewFactory struct{}

func (i *ListUnspentTokensViewFactory) NewView(in []byte) (view.View, error) {
	f := &ListUnspentTokensView{ListUnspentTokens: &ListUnspentTokens{}}
	err := json.Unmarshal(in, f.ListUnspentTokens)
	assert.NoError(err, "failed unmarshalling input")
	return f, nil
}
