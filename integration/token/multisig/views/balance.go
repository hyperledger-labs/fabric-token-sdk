/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package views

import (
	"encoding/json"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/multisig"
	"strconv"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type BalanceQuery struct {
	TMSID  *token.TMSID
	Wallet string
	Type   token2.Type
}

type BalanceResult struct {
	Type     token2.Type
	Quantity string
	CoOwned  string
}

// BalanceView is a view used to return:
// 1. The amount of unspent tokens;
// 2. The amount of co-owned tokens;

type BalanceView struct {
	*BalanceQuery
}

func (b *BalanceView) Call(context view.Context) (interface{}, error) {
	var tms *token.ManagementService
	if b.TMSID != nil {
		tms = token.GetManagementService(context, token.WithTMSID(*b.TMSID))
	} else {
		tms = token.GetManagementService(context)
	}
	wallet := tms.WalletManager().OwnerWallet(b.Wallet)
	assert.NotNil(wallet, "failed to get wallet [%s]", b.Wallet)
	precision := tms.PublicParametersManager().PublicParameters().Precision()

	// owned
	balance, err := wallet.Balance(token.WithType(b.Type))
	assert.NoError(err, "failed to get unspent tokens")

	escrowWallet := multisig.Wallet(context, wallet)
	// co-owned
	coOwnedTokens, err := escrowWallet.ListTokensIterator(token.WithType(b.Type))
	assert.NoError(err, "failed to get co-owned tokens")
	coOwned, err := coOwnedTokens.Sum(precision)
	assert.NoError(err, "failed to compute the sum of the co-owned tokens")

	return BalanceResult{
		Quantity: strconv.FormatUint(balance, 10),
		CoOwned:  coOwned.Decimal(),
		Type:     b.Type,
	}, nil
}

type BalanceViewFactory struct{}

func (g *BalanceViewFactory) NewView(in []byte) (view.View, error) {
	f := &BalanceView{BalanceQuery: &BalanceQuery{}}
	if err := json.Unmarshal(in, f.BalanceQuery); err != nil {
		return nil, err
	}
	return f, nil
}
