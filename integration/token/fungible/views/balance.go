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
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx/multisig"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
)

type BalanceQuery struct {
	TMSID     *token.TMSID
	SkipCheck bool
	Wallet    string
	Type      token2.Type
}

type Balance struct {
	Type     token2.Type
	Quantity string
	CoOwned  string
}

type BalanceView struct {
	*BalanceQuery
}

func (b *BalanceView) Call(context view.Context) (interface{}, error) {
	span := trace.SpanFromContext(context.Context())
	defer span.AddEvent("end_balance_view")
	span.AddEvent("start_balance_view")
	tms := token.GetManagementService(context, ServiceOpts(b.TMSID)...)
	wallet := tms.WalletManager().OwnerWallet(b.Wallet)
	if wallet == nil {
		return nil, fmt.Errorf("wallet %s not found", b.Wallet)
	}

	span.AddEvent("start_sum_calculation_unspent")
	unspentTokens, err := wallet.ListUnspentTokens(token.WithType(b.Type))
	if err != nil {
		return nil, errors.Wrapf(err, "failed listing unspent tokens")
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
	span.AddEvent("end_sum_calculation_unspent")

	// co-owned
	multisigWallet := multisig.Wallet(context, wallet)
	coOwnedTokens, err := multisigWallet.ListTokensIterator(token.WithType(b.Type))
	assert.NoError(err, "failed to get co-owned tokens")
	coOwned, err := multisig.Sum(coOwnedTokens, precision)
	assert.NoError(err, "failed to compute the sum of the co-owned tokens")

	if !b.SkipCheck {
		span.AddEvent("start_sum_calculation")
		balance, err := wallet.Balance(token.WithType(b.Type))
		if err != nil {
			return nil, err
		}
		span.AddEvent("end_sum_calculation")
		if sum.ToBigInt().Uint64() != balance {
			return nil, errors.Errorf("balance doesn't match [%d]!=[%d]", balance, sum.ToBigInt().Uint64())
		}
	}

	return Balance{
		Quantity: sum.Decimal(),
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

type CoOwnedBalanceQuery struct {
	TMSID  *token.TMSID
	Wallet string
	Type   token2.Type
}

type CoOwnedBalanceView struct {
	*CoOwnedBalanceQuery
}

func (b *CoOwnedBalanceView) Call(context view.Context) (interface{}, error) {
	tms := token.GetManagementService(context, ServiceOpts(b.TMSID)...)
	wallet := tms.WalletManager().OwnerWallet(b.Wallet)
	if wallet == nil {
		return nil, fmt.Errorf("wallet %s not found", b.Wallet)
	}

	// co-owned
	precision := tms.PublicParametersManager().PublicParameters().Precision()
	multisigWallet := multisig.Wallet(context, wallet)
	coOwnedTokens, err := multisigWallet.ListTokensIterator(token.WithType(b.Type))
	assert.NoError(err, "failed to get co-owned tokens")
	coOwned, err := multisig.Sum(coOwnedTokens, precision)
	assert.NoError(err, "failed to compute the sum of the co-owned tokens")

	return Balance{
		Quantity: coOwned.Decimal(),
		Type:     b.Type,
	}, nil
}

type CoOwnedBalanceViewFactory struct{}

func (g *CoOwnedBalanceViewFactory) NewView(in []byte) (view.View, error) {
	f := &CoOwnedBalanceView{CoOwnedBalanceQuery: &CoOwnedBalanceQuery{}}
	if err := json.Unmarshal(in, f.CoOwnedBalanceQuery); err != nil {
		return nil, err
	}
	return f, nil
}
