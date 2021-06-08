/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package views

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxcc"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type Transfer struct {
	Wallet    string
	TokenIDs  []*token.Id
	Type      string
	Amount    uint64
	Recipient view.Identity
	Retry     bool
}

type TransferView struct {
	*Transfer
}

func (t *TransferView) Call(context view.Context) (interface{}, error) {
	recipient, err := ttxcc.RequestRecipientIdentity(context, t.Recipient)
	assert.NoError(err, "failed getting recipient")

	// Prepare transaction
	tx, err := ttxcc.NewAnonymousTransaction(
		context,
		ttxcc.WithAuditor(fabric.GetIdentityProvider(context).Identity("auditor")),
	)
	assert.NoError(err, "failed creating transaction")

	err = tx.Transfer(
		ttxcc.GetWallet(context, t.Wallet),
		t.Type,
		[]uint64{t.Amount},
		[]view.Identity{recipient},
		token2.WithTokenIDs(t.TokenIDs...),
	)
	assert.NoError(err, "failed adding new tokens")

	_, err = context.RunView(ttxcc.NewCollectEndorsementsView(tx))
	assert.NoError(err, "failed to sign transaction")

	// Send to the ordering service and wait for confirmation
	_, err = context.RunView(ttxcc.NewOrderingView(tx))
	assert.NoError(err, "failed asking ordering")

	return tx.ID(), nil
}

type TransferViewFactory struct{}

func (p *TransferViewFactory) NewView(in []byte) (view.View, error) {
	f := &TransferView{Transfer: &Transfer{}}
	err := json.Unmarshal(in, f.Transfer)
	assert.NoError(err, "failed unmarshalling input")
	return f, nil
}

type TransferWithSelectorView struct {
	*Transfer
}

func (t *TransferWithSelectorView) Call(context view.Context) (interface{}, error) {
	recipient, err := ttxcc.RequestRecipientIdentity(context, t.Recipient)
	assert.NoError(err, "failed getting recipient")

	// Prepare transaction
	tx, err := ttxcc.NewAnonymousTransaction(
		context,
		ttxcc.WithAuditor(fabric.GetIdentityProvider(context).Identity("auditor")),
	)
	assert.NoError(err, "failed creating transaction")

	if len(t.TokenIDs) == 0 {
		selector, err := tx.Selector()
		assert.NoError(err, "failed getting token selector")

		var ids []*token.Id
		var sum token.Quantity

		for i := 0; i < 5; i++ {
			ids, sum, err = selector.Select(
				ttxcc.GetWallet(context, t.Wallet),
				token.NewQuantityFromUInt64(t.Amount).Decimal(),
				t.Type,
			)
			if err != nil && t.Retry {
				time.Sleep(10 * time.Second)
				continue
			}
			break
		}
		if err != nil {
			cause := errors.Cause(err)
			switch cause {
			case nil:
				assert.NoError(err, "system failure")
			case token2.SelectorInsufficientFunds:
				assert.NoError(err, "pineapple")
			case token2.SelectorSufficientButLockedFunds:
				assert.NoError(err, "lemonade")
			case token2.SelectorSufficientButNotCertifiedFunds:
				assert.NoError(err, "mandarine")
			}
		}

		// do some check on the selected outputs
		tokens, err := tx.TokenService().Vault().NewQueryEngine().GetTokens(ids...)
		assert.NoError(err, "failed getting tokens from ids")

		recomputedSum := token.NewQuantityFromUInt64(0)
		for _, tok := range tokens {
			assert.Equal(t.Type, tok.Type, "expected token of type [%s], got [%s]", t.Type, tok.Type)
			q, err := token.ToQuantity(tok.Quantity, 64)
			assert.NoError(err, "failed converting quantity")
			recomputedSum = recomputedSum.Add(q)
		}
		assert.True(sum.Cmp(recomputedSum) == 0, "sums do not match")
		assert.False(sum.Cmp(token.NewQuantityFromUInt64(t.Amount)) < 0, "if this point is reached, funds are sufficients")

		t.TokenIDs = ids
	}

	err = tx.Transfer(
		ttxcc.GetWallet(context, t.Wallet),
		t.Type,
		[]uint64{t.Amount},
		[]view.Identity{recipient},
		token2.WithTokenIDs(t.TokenIDs...),
	)
	assert.NoError(err, "failed adding new tokens")

	_, err = context.RunView(ttxcc.NewCollectEndorsementsView(tx))
	assert.NoError(err, "failed to sign transaction")

	if !t.Retry {
		// Introduce a delay that will keep the tokens locked by the selector
		time.Sleep(20 * time.Second)
	}

	// Send to the ordering service and wait for confirmation
	_, err = context.RunView(ttxcc.NewOrderingView(tx))
	assert.NoError(err, "failed asking ordering")

	return tx.ID(), nil
}

type TransferWithSelectorViewFactory struct{}

func (p *TransferWithSelectorViewFactory) NewView(in []byte) (view.View, error) {
	f := &TransferWithSelectorView{Transfer: &Transfer{}}
	err := json.Unmarshal(in, f.Transfer)
	assert.NoError(err, "failed unmarshalling input")
	return f, nil
}
