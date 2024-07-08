/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package views

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type BalanceQuery struct {
	TMSID  token.TMSID
	Wallet string
	Type   string
}

type Balance struct {
	Type     string
	Quantity string
}

type BalanceView struct {
	*BalanceQuery
}

func (b *BalanceView) Call(context view.Context) (interface{}, error) {
	tms := token.GetManagementService(context, token.WithTMSID(b.TMSID))
	wallet := tms.WalletManager().OwnerWallet(b.Wallet)
	if wallet == nil {
		return nil, fmt.Errorf("wallet %s not found", b.Wallet)
	}

	unspentTokens, err := wallet.ListUnspentTokens(context.Context(), token.WithType(b.Type))
	if err != nil {
		return nil, err
	}

	precision := tms.PublicParametersManager().PublicParameters().Precision()
	sum := token2.NewZeroQuantity(precision)
	for _, tok := range unspentTokens.Tokens {
		q, err := token2.ToQuantity(tok.Quantity, precision)
		if err != nil {
			return nil, err
		}
		sum = sum.Add(q)
	}

	return Balance{Quantity: sum.Decimal(), Type: b.Type}, nil
}

type BalanceViewFactory struct{}

func (g *BalanceViewFactory) NewView(in []byte) (view.View, error) {
	f := &BalanceView{BalanceQuery: &BalanceQuery{}}
	if err := json.Unmarshal(in, f.BalanceQuery); err != nil {
		return nil, err
	}
	return f, nil
}
