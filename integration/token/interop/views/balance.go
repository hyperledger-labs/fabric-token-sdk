/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package views

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/htlc"
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

// BalanceView is a view used to return:
// 1. The amount of unspent tokens;
// 2. The amount of htlc-locked tokens not yet expired;
// 3. The amount of expired htlc-locked tokens that have been not reclaimed
// for the given wallet.
type BalanceView struct {
	*Balance
}

func (b *BalanceView) Call(context view.Context) (interface{}, error) {
	tms := token.GetManagementService(context, token.WithTMSID(b.TMSID))
	wallet := tms.WalletManager().OwnerWallet(b.Wallet)
	assert.NotNil(wallet, "failed to get wallet [%s]", b.Wallet)

	// owned
	unspentTokens, err := wallet.ListUnspentTokensIterator(token.WithType(b.Type))
	assert.NoError(err, "failed to get unspent tokens")
	precision := tms.PublicParametersManager().PublicParameters().Precision()
	ownedSum, err := unspentTokens.Sum(precision)
	assert.NoError(err, "failed to compute the sum of the unspent tokens")

	htlcWallet := htlc.Wallet(context, wallet)
	// locked
	lockedToTokens, err := htlcWallet.ListTokensIterator(token.WithType(b.Type))
	assert.NoError(err, "failed to get locked tokens")
	lockedSum, err := lockedToTokens.Sum(precision)
	assert.NoError(err, "failed to compute the sum of the htlc locked tokens")

	// expired
	err = htlcWallet.DeleteExpiredReceivedTokens(context, token.WithType(b.Type))
	assert.NoError(err, "failed to delete expired tokens")
	expiredTokens, err := htlcWallet.ListExpiredReceivedTokensIterator(token.WithType(b.Type))
	assert.NoError(err, "failed to get expired tokens")
	expiredSum, err := expiredTokens.Sum(precision)
	assert.NoError(err, "failed to compute the sum of the htlc expired tokens")

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
