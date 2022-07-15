/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package query

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

var logger = flogging.MustGetLogger("token-sdk.tms.zkat.balance")

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

	unspentTokens, err := wallet.ListUnspentTokens(token.WithType(b.Type))
	if err != nil {
		return nil, err
	}

	precision := tms.PublicParametersManager().Precision()
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

type AllMyBalances struct {
	Balances []Balance
}

type AllBalanceQuery struct {
	TMSID  token.TMSID
	Wallet string
}

type AllMyBalanceView struct {
	*AllBalanceQuery
}

func (b *AllMyBalanceView) Call(context view.Context) (interface{}, error) {
	tms := token.GetManagementService(context, token.WithTMSID(b.TMSID))
	wallet := tms.WalletManager().OwnerWallet(b.Wallet)
	if wallet == nil {
		return nil, fmt.Errorf("wallet %s not found", b.Wallet)
	}

	balances := make(map[string]token2.Quantity)
	unspentTokens, err := wallet.ListUnspentTokens()
	if err != nil {
		return nil, err
	}
	precision := tms.PublicParametersManager().Precision()
	for _, tok := range unspentTokens.Tokens {
		fmt.Printf("\n am I here? \n")
		_, exists := balances[tok.Type]
		if !exists {
			balances[tok.Type] = token2.NewZeroQuantity(precision)
		}
		q, err := token2.ToQuantity(tok.Quantity, precision)
		if err != nil {
			return nil, err
		}
		balances[tok.Type] = balances[tok.Type].Add(q)
	}
	var mybalance []Balance
	for k := range balances {
		mybalance = append(mybalance, Balance{Type: k, Quantity: balances[k].Decimal()})
	}

	return AllMyBalances{mybalance}, nil
}

type AllMyBalanceViewFactory struct{}

func (g *AllMyBalanceViewFactory) NewView(in []byte) (view.View, error) {
	f := &AllMyBalanceView{AllBalanceQuery: &AllBalanceQuery{}}
	if err := json.Unmarshal(in, f.AllBalanceQuery); err != nil {
		return nil, err
	}
	return f, nil
}
