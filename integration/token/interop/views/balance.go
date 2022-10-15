/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package views

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/htlc"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type Balance struct {
	TMSID  token.TMSID
	Wallet string
	Type   string
}

type BalanceResult struct {
	Type     string
	Quantity string
	Locked   string
	Expired  string
}

type BalanceView struct {
	*Balance
}

func (b *BalanceView) Call(context view.Context) (interface{}, error) {
	tms := token.GetManagementService(context, token.WithTMSID(b.TMSID))
	wallet := tms.WalletManager().OwnerWallet(b.Wallet)
	if wallet == nil {
		return nil, fmt.Errorf("wallet %s not found", b.Wallet)
	}

	// owned
	unspentTokens, err := wallet.ListUnspentTokens(token.WithType(b.Type))
	if err != nil {
		return nil, err
	}
	precision := tms.PublicParametersManager().Precision()
	ownedSum := token2.NewZeroQuantity(precision)
	for _, tok := range unspentTokens.Tokens {
		q, err := token2.ToQuantity(tok.Quantity, precision)
		if err != nil {
			return nil, err
		}
		ownedSum = ownedSum.Add(q)
	}

	// locked
	lockedToTokens, err := htlc.Wallet(context, wallet).ListTokens(token.WithType(b.Type))
	assert.NoError(err, "failed to get locked to tokens")
	lockedSum := token2.NewZeroQuantity(precision)
	for _, tok := range lockedToTokens.Tokens {
		q, err := token2.ToQuantity(tok.Quantity, precision)
		if err != nil {
			return nil, err
		}
		lockedSum = lockedSum.Add(q)
	}

	// expired
	expiredTokens, err := htlc.Wallet(context, wallet).CleanupExpiredReceivedTokens(context, token.WithType(b.Type))
	assert.NoError(err, "failed to get expired tokens")
	expiredSum := token2.NewZeroQuantity(precision)
	for _, tok := range expiredTokens.Tokens {
		q, err := token2.ToQuantity(tok.Quantity, precision)
		if err != nil {
			return nil, err
		}
		expiredSum = expiredSum.Add(q)
	}

	return BalanceResult{
		Quantity: ownedSum.Decimal(),
		Locked:   lockedSum.Decimal(),
		Expired:  expiredSum.Decimal(),
		Type:     b.Type,
	}, nil
}

type BalanceViewFactory struct{}

func (g *BalanceViewFactory) NewView(in []byte) (view.View, error) {
	f := &BalanceView{Balance: &Balance{}}
	if err := json.Unmarshal(in, f.Balance); err != nil {
		return nil, err
	}
	return f, nil
}
