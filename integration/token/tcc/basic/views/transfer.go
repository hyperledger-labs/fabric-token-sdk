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
	TokenIDs []*token.ID
	// Type of tokens to transfer
	Type string
	// Amount to transfer
	Amount uint64
	// Recipient is the identity of the recipient's FSC node
	Recipient view.Identity
	// Retry tells if a retry must happen in case of a failure
	Retry bool
	// FailToRelease if true, it fails after transfer to trigger the Release function on the token transaction
	FailToRelease bool
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
		ttxcc.WithAuditor(fabric.GetDefaultIdentityProvider(context).Identity("auditor")),
	)
	assert.NoError(err, "failed creating transaction")

	// The sender will select tokens owned by this wallet
	senderWallet := ttxcc.GetWallet(context, t.Wallet)
	assert.NotNil(senderWallet, "sender wallet [%s] not found", t.Wallet)

	// The sender adds a new transfer operation to the transaction following the instruction received.
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

	if t.FailToRelease {
		// return an error to trigger the Release function on the token transaction
		// The Release function is called when the context is canceled due to a panic or an error.
		return nil, errors.New("test release")
	}

	// The sender is ready to collect all the required signatures.
	// In this case, the sender's and the auditor's signatures.
	// Invoke the Token Chaincode to collect endorsements on the Token Request and prepare the relative Fabric transaction.
	// This is all done in one shot running the following view.
	// Before completing, all recipients receive the approved Fabric transaction.
	// Depending on the token driver implementation, the recipient's signature might or might not be needed to make
	// the token transaction valid.
	_, err = context.RunView(ttxcc.NewCollectEndorsementsView(tx))
	assert.NoError(err, "failed to sign transaction")

	// Sanity checks:
	// - the transaction is in busy state in the vault
	fns := fabric.GetFabricNetworkService(context, tx.Network())
	ch, err := fns.Channel(tx.Channel())
	assert.NoError(err, "failed to retrieve channel [%s]", tx.Channel())
	vc, _, err := ch.Vault().Status(tx.ID())
	assert.NoError(err, "failed to retrieve vault status for transaction [%s]", tx.ID())
	assert.Equal(fabric.Busy, vc, "transaction [%s] should be in busy state", tx.ID())
	vc, _, err = ch.Committer().Status(tx.ID())
	assert.NoError(err, "failed to retrieve vault status for transaction [%s]", tx.ID())
	assert.Equal(fabric.Busy, vc, "transaction [%s] should be in busy state", tx.ID())

	// Send to the ordering service and wait for finality
	_, err = context.RunView(ttxcc.NewOrderingAndFinalityView(tx))
	assert.NoError(err, "failed asking ordering")

	// Sanity checks:
	// - the transaction is in valid state in the vault
	vc, _, err = ch.Vault().Status(tx.ID())
	assert.NoError(err, "failed to retrieve vault status for transaction [%s]", tx.ID())
	assert.Equal(fabric.Valid, vc, "transaction [%s] should be in valid state", tx.ID())
	vc, _, err = ch.Committer().Status(tx.ID())
	assert.NoError(err, "failed to retrieve vault status for transaction [%s]", tx.ID())
	assert.Equal(fabric.Valid, vc, "transaction [%s] should be in busy state", tx.ID())

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
		ttxcc.WithAuditor(fabric.GetDefaultIdentityProvider(context).Identity("auditor")),
	)
	assert.NoError(err, "failed creating transaction")

	// The sender will select tokens owned by this wallet
	senderWallet := ttxcc.GetWallet(context, t.Wallet)
	assert.NotNil(senderWallet, "sender wallet [%s] not found", t.Wallet)

	// If no specific tokens are requested, then a custom token selection process start
	if len(t.TokenIDs) == 0 {
		// The sender uses the default token selector each transaction comes equipped with
		selector, err := tx.Selector()
		assert.NoError(err, "failed getting token selector")

		// The sender tries to select the requested amount of tokens of the passed type.
		// If a failure happens, the sender retries up to 5 times, waiting 10 seconds after each failure.
		// This is just an example, any other policy can be implemented.
		var ids []*token.ID
		var sum token.Quantity

		for i := 0; i < 5; i++ {
			// Select the request amount of tokens of the given type
			ids, sum, err = selector.Select(
				ttxcc.GetWallet(context, t.Wallet),
				token.NewQuantityFromUInt64(t.Amount).Decimal(),
				t.Type,
			)
			// If an error occurs and retry has been asked, then wait first a bit
			if err != nil && t.Retry {
				time.Sleep(10 * time.Second)
				continue
			}
			break
		}
		if err != nil {
			// If finally not enough tokens were available, the sender can check what was the cause of the error:
			cause := errors.Cause(err)
			switch cause {
			case nil:
				assert.NoError(err, "system failure")
			case token2.SelectorInsufficientFunds:
				assert.NoError(err, "pineapple")
			case token2.SelectorSufficientButLockedFunds:
				assert.NoError(err, "lemonade")
			case token2.SelectorSufficientButNotCertifiedFunds:
				assert.NoError(err, "mandarin")
			case token2.SelectorSufficientFundsButConcurrencyIssue:
				assert.NoError(err, "peach")
			}
		}

		// If the sender reaches this point, it means that tokens are available.
		// The sender can further inspect these tokens, if the business logic requires to do so.
		// Here is an example. The sender double checks that the tokens selected are the expected

		// First, the sender queries the vault to get the tokens
		tokens, err := tx.TokenService().Vault().NewQueryEngine().GetTokens(ids...)
		assert.NoError(err, "failed getting tokens from ids")

		// Then, the sender double check that what returned by the selector is correct
		recomputedSum := token.NewQuantityFromUInt64(0)
		for _, tok := range tokens {
			// Is the token of the right type?
			assert.Equal(t.Type, tok.Type, "expected token of type [%s], got [%s]", t.Type, tok.Type)
			// Add the quantity to the total
			q, err := token.ToQuantity(tok.Quantity, 64)
			assert.NoError(err, "failed converting quantity")
			recomputedSum = recomputedSum.Add(q)
		}
		// Is the recomputed sum correct?
		assert.True(sum.Cmp(recomputedSum) == 0, "sums do not match")
		// Is the amount selected equal or larger than what requested?
		assert.False(sum.Cmp(token.NewQuantityFromUInt64(t.Amount)) < 0, "if this point is reached, funds are sufficients")

		t.TokenIDs = ids
	}

	// The sender adds a new transfer operation to the transaction following the instruction received.
	// Notice the use of `token2.WithTokenIDs(t.TokenIDs...)` to pass the token ids selected above.
	err = tx.Transfer(
		ttxcc.GetWallet(context, t.Wallet),
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

	// Sanity checks:
	// - the transaction is in busy state in the vault
	fns := fabric.GetFabricNetworkService(context, tx.Network())
	ch, err := fns.Channel(tx.Channel())
	assert.NoError(err, "failed to retrieve channel [%s]", tx.Channel())
	vc, _, err := ch.Vault().Status(tx.ID())
	assert.NoError(err, "failed to retrieve vault status for transaction [%s]", tx.ID())
	assert.Equal(fabric.Busy, vc, "transaction [%s] should be in busy state", tx.ID())
	vc, _, err = ch.Committer().Status(tx.ID())
	assert.NoError(err, "failed to retrieve vault status for transaction [%s]", tx.ID())
	assert.Equal(fabric.Busy, vc, "transaction [%s] should be in busy state", tx.ID())

	if !t.Retry {
		// Introduce a delay that will keep the tokens locked by the selector
		time.Sleep(20 * time.Second)
	}

	// Send to the ordering service and wait for finality
	_, err = context.RunView(ttxcc.NewOrderingAndFinalityView(tx))
	assert.NoError(err, "failed asking ordering")

	// Sanity checks:
	// - the transaction is in valid state in the vault
	vc, _, err = ch.Vault().Status(tx.ID())
	assert.NoError(err, "failed to retrieve vault status for transaction [%s]", tx.ID())
	assert.Equal(fabric.Valid, vc, "transaction [%s] should be in valid state", tx.ID())
	vc, _, err = ch.Committer().Status(tx.ID())
	assert.NoError(err, "failed to retrieve vault status for transaction [%s]", tx.ID())
	assert.Equal(fabric.Valid, vc, "transaction [%s] should be in busy state", tx.ID())

	return tx.ID(), nil
}

type TransferWithSelectorViewFactory struct{}

func (p *TransferWithSelectorViewFactory) NewView(in []byte) (view.View, error) {
	f := &TransferWithSelectorView{Transfer: &Transfer{}}
	err := json.Unmarshal(in, f.Transfer)
	assert.NoError(err, "failed unmarshalling input")
	return f, nil
}
