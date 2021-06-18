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

// Transfer contains the input information for a transfer
type Transfer struct {
	// Wallet is the identifier of the wallet that owns the tokens to transfer
	Wallet string
	// TokenIDs contains a list of token ids to transfer. If empty, tokens are selected on the spot.
	TokenIDs []*token.Id
	// Type of tokens to transfer
	Type string
	// Amount to transfer
	Amount uint64
	// Recipient is the identity of the recipient's FSC node
	Recipient view.Identity
	// Retry tells if a retry must happen in case of a failure
	Retry bool
}

type TransferView struct {
	*Transfer
}

func (t *TransferView) Call(context view.Context) (interface{}, error) {
	// As a first step operation, the sender contacts the recipient's FSC node
	// to ask for the identity to use to assign ownership of the freshly created token.
	// Notice that, this step would not be required if the sender knew already which
	// identity the recipient wants to use.
	recipient, err := ttxcc.RequestRecipientIdentity(context, t.Recipient)
	assert.NoError(err, "failed getting recipient")

	// At this point, the sender is ready to prepare the token transaction.
	// The sender creates an anonymous transaction (this means that the result Fabric transaction will be signed using idemix),
	// and specify the auditor that must be contacted to approve the operation.
	tx, err := ttxcc.NewAnonymousTransaction(
		context,
		ttxcc.WithAuditor(fabric.GetIdentityProvider(context).Identity("auditor")),
	)
	assert.NoError(err, "failed creating transaction")

	senderWallet := ttxcc.GetWallet(context, t.Wallet)
	assert.NotNil(senderWallet, "sender wallet [%s] not found", t.Wallet)

	// The senders adds a new transfer operation to the transaction following the instruction received.
	// Notice the use of `token2.WithTokenIDs(t.TokenIDs...)`. If t.TokenIDs is not empty, the Transfer
	// function uses those tokens, otherwise the tokens will be selected on the spot.
	// Token selection happens internally by invoking the default token selector:
	// selector, err := tx.TokenService().SelectorManager().NewSelector(tx.ID())
	// assert.NoError(err, "failed getting selector")
	// selector.Select(wallet, amount, tokenType)
	// It is also possible to pass a custom token selector to the Transfer function by using the relative opt:
	// token2.WithTokenSelector(selector).
	err = tx.Transfer(
		senderWallet,
		t.Type,
		[]uint64{t.Amount},
		[]view.Identity{recipient},
		token2.WithTokenIDs(t.TokenIDs...),
	)
	assert.NoError(err, "failed adding new tokens")

	// The sender is ready to collect all the required signatures.
	// In this case, the sender's and the auditor's signatures.
	// Invoke the Token Chaincode to collect endorsements on the Token Request and prepare the relative Fabric transaction.
	// This is all done in one shot running the following view.
	// Before completing, all recipients receive the approved Fabric transaction.
	// Depending on the token driver implementation, the recipient's signature might or might not be needed to make
	// the token transaction valid.
	_, err = context.RunView(ttxcc.NewCollectEndorsementsView(tx))
	assert.NoError(err, "failed to sign transaction")

	// Send to the ordering service and wait for finality
	_, err = context.RunView(ttxcc.NewOrderingAndFinalityView(tx))
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
	_, err = context.RunView(ttxcc.NewOrderingAndFinalityView(tx))
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
