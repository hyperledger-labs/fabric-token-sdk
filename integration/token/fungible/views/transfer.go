/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package views

import (
	"encoding/json"
	"strings"
	"time"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	view4 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

// TransferAction defines a transfer action
type TransferAction struct {
	// Amount to transfer
	Amount uint64
	// Recipient is the identity of the recipient's FSC node
	Recipient view.Identity
	// RecipientEID is the expected enrolment id of the recipient
	RecipientEID string
}

// Transfer contains the input information for a transfer
type Transfer struct {
	// Auditor is the name of the auditor that must be contacted to approve the operation
	Auditor string
	// Wallet is the identifier of the wallet that owns the tokens to transfer
	Wallet         string
	ExternalWallet bool
	// TokenIDs contains a list of token ids to transfer. If empty, tokens are selected on the spot.
	TokenIDs []*token.ID
	// Type of tokens to transfer
	Type string
	// Amount to transfer
	Amount uint64
	// Recipient is the identity of the recipient's FSC node
	Recipient view.Identity
	// RecipientEID is the expected enrolment id of the recipient
	RecipientEID string
	// Retry tells if a retry must happen in case of a failure
	Retry bool
	// FailToRelease if true, it fails after transfer to trigger the Release function on the token transaction
	FailToRelease bool
	// For additional transfer actions
	TransferAction []TransferAction

	RecipientData *token2.RecipientData
	// The TMS to pick in case of multiple TMSIDs
	TMSID *token2.TMSID
}

type TransferView struct {
	*Transfer
}

func (t *TransferView) Call(context view.Context) (txID interface{}, err error) {
	// As a first step operation, the sender contacts the recipient's FSC node
	// to ask for the identity to use to assign ownership of the freshly created token.
	// Notice that, this step would not be required if the sender knew already which
	// identity the recipient wants to use.
	recipient, err := ttx.RequestRecipientIdentity(context, t.Recipient, ServiceOpts(t.TMSID)...)
	assert.NoError(err, "failed getting recipient")

	wm := token2.GetManagementService(context, ServiceOpts(t.TMSID)...).WalletManager()
	// if there are more recipients, ask for their recipient identity
	var additionalRecipients []view.Identity
	for _, action := range t.TransferAction {
		actionRecipient, err := ttx.RequestRecipientIdentity(context, action.Recipient, ServiceOpts(t.TMSID)...)
		assert.NoError(err, "failed getting recipient")
		eID, err := wm.GetEnrollmentID(recipient)
		assert.NoError(err, "failed to get enrollment id for recipient [%s]", recipient)
		assert.True(strings.HasPrefix(eID, t.RecipientEID), "recipient EID [%s] does not match the expected one [%s]", eID, t.RecipientEID)
		additionalRecipients = append(additionalRecipients, actionRecipient)
	}

	// match recipient EID
	eID, err := wm.GetEnrollmentID(recipient)
	assert.NoError(err, "failed to get enrollment id for recipient [%s]", recipient)
	assert.True(strings.HasPrefix(eID, t.RecipientEID), "recipient EID [%s] does not match the expected one [%s]", eID, t.RecipientEID)

	// At this point, the sender is ready to prepare the token transaction.
	// The sender creates an anonymous transaction (this means that the resulting Fabric transaction will be signed using idemix, for example),
	// and specify the auditor that must be contacted to approve the operation.
	tx, err := ttx.NewAnonymousTransaction(
		context,
		append(TxOpts(t.TMSID), ttx.WithAuditor(view2.GetIdentityProvider(context).Identity(t.Auditor)))...,
	)
	assert.NoError(err, "failed creating transaction")

	// The sender will select tokens owned by this wallet
	senderWallet := ttx.GetWallet(context, t.Wallet, ServiceOpts(t.TMSID)...)
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
		token2.WithRestRecipientIdentity(t.RecipientData),
	)
	assert.NoError(err, "failed adding transfer action [%d:%s]", t.Amount, t.Recipient)

	// add additional transfers
	for i, action := range t.TransferAction {
		err = tx.Transfer(
			senderWallet,
			t.Type,
			[]uint64{action.Amount},
			[]view.Identity{additionalRecipients[i]},
			token2.WithTokenIDs(t.TokenIDs...),
		)
		assert.NoError(err, "failed adding transfer action [%d:%s]", action.Amount, action.Recipient)
	}

	if t.FailToRelease {
		// return an error to trigger the Release function on the token transaction
		// The Release function is called when the context is canceled due to a panic or an error.
		return nil, errors.New("test release")
	}

	// The sender is ready to collect all the required signatures.
	// In this case, the sender's and the auditor's signatures.
	// Invoke the Token Chaincode to collect endorsements on the Token Request and prepare the relative transaction.
	// This is all done in one shot running the following view.
	// Before completing, all recipients receive the approved transaction.
	// Depending on the token driver implementation, the recipient's signature might or might not be needed to make
	// the token transaction valid.
	var endorserOpts []ttx.EndorsementsOpt
	if t.ExternalWallet {
		// if ExternalWallet is set to true, then the signatures that the wallet must generate are prepared externally to this FSC node.
		// Here, we assume that the view has been called using GRPC stream
		stream := view4.GetStream(context)
		endorserOpts = append(endorserOpts, ttx.WithExternalWalletSigner(t.Wallet, ttx.NewStreamExternalWalletSignerServer(stream)))
	}
	_, err = context.RunView(ttx.NewCollectEndorsementsView(tx, endorserOpts...))
	assert.NoError(err, "failed to sign transaction [<<<%s>>>]", tx.ID())

	// Sanity checks:
	// - the transaction is in busy state in the vault
	net := network.GetInstance(context, tx.Network(), tx.Channel())
	vault, err := net.Vault(tx.Namespace())
	assert.NoError(err, "failed to retrieve vault [%s]", tx.Namespace())
	vc, err := vault.Status(tx.ID())
	assert.NoError(err, "failed to retrieve vault status for transaction [%s]", tx.ID())
	assert.Equal(network.Busy, vc, "transaction [%s] should be in busy state", tx.ID())

	// Send to the ordering service and wait for finality
	_, err = context.RunView(ttx.NewOrderingAndFinalityView(tx))
	assert.NoError(err, "failed asking ordering")

	// Sanity checks:
	// - the transaction is in valid state in the vault
	vc, err = vault.Status(tx.ID())
	assert.NoError(err, "failed to retrieve vault status for transaction [%s]", tx.ID())
	assert.Equal(network.Valid, vc, "transaction [%s] should be in valid state", tx.ID())

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
	recipient, err := ttx.RequestRecipientIdentity(context, t.Recipient)
	assert.NoError(err, "failed getting recipient")

	// At this point, the sender is ready to prepare the token transaction.
	// The sender creates an anonymous transaction (this means that the resulting Fabric transaction will be signed using idemix, for example),
	// and specify the auditor that must be contacted to approve the operation.
	tx, err := ttx.NewAnonymousTransaction(
		context,
		ttx.WithAuditor(view2.GetIdentityProvider(context).Identity(t.Auditor)),
	)
	assert.NoError(err, "failed creating transaction")

	// The sender will select tokens owned by this wallet
	senderWallet := ttx.GetWallet(context, t.Wallet)
	assert.NotNil(senderWallet, "sender wallet [%s] not found", t.Wallet)

	// If no specific tokens are requested, then a custom token selection process start
	precision := token2.GetManagementService(context).PublicParametersManager().PublicParameters().Precision()
	amount, err := token.UInt64ToQuantity(t.Amount, precision)
	assert.NoError(err, "failed to convert to quantity")

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
				ttx.GetWallet(context, t.Wallet),
				amount.Decimal(),
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
		recomputedSum := token.NewZeroQuantity(precision)
		for _, tok := range tokens {
			// Is the token of the right type?
			assert.Equal(t.Type, tok.Type, "expected token of type [%s], got [%s]", t.Type, tok.Type)
			// Add the quantity to the total
			q, err := token.ToQuantity(tok.Quantity, precision)
			assert.NoError(err, "failed converting quantity")
			recomputedSum = recomputedSum.Add(q)
		}
		// Is the recomputed sum correct?
		assert.True(sum.Cmp(recomputedSum) == 0, "sums do not match")
		// Is the amount selected equal or larger than what requested?
		assert.False(sum.Cmp(amount) < 0, "if this point is reached, funds are sufficient")

		t.TokenIDs = ids
	}

	// The sender adds a new transfer operation to the transaction following the instruction received.
	// Notice the use of `token2.WithTokenIDs(t.TokenIDs...)` to pass the token ids selected above.
	err = tx.Transfer(
		ttx.GetWallet(context, t.Wallet),
		t.Type,
		[]uint64{t.Amount},
		[]view.Identity{recipient},
		token2.WithTokenIDs(t.TokenIDs...),
	)
	assert.NoError(err, "failed adding new tokens")

	// The sender is ready to collect all the required signatures.
	// In this case, the sender's and the auditor's signatures.
	// Invoke the Token Chaincode to collect endorsements on the Token Request and prepare the relative transaction.
	// This is all done in one shot running the following view.
	// Before completing, all recipients receive the approved transaction.
	// Depending on the token driver implementation, the recipient's signature might or might not be needed to make
	// the token transaction valid.
	_, err = context.RunView(ttx.NewCollectEndorsementsView(tx))
	assert.NoError(err, "failed to sign transaction")

	// Sanity checks:
	// - the transaction is in busy state in the vault
	net := network.GetInstance(context, tx.Network(), tx.Channel())
	vault, err := net.Vault(tx.Namespace())
	assert.NoError(err, "failed to retrieve vault [%s]", tx.Namespace())
	vc, err := vault.Status(tx.ID())
	assert.NoError(err, "failed to retrieve vault status for transaction [%s]", tx.ID())
	assert.Equal(network.Busy, vc, "transaction [%s] should be in busy state", tx.ID())

	if !t.Retry {
		// Introduce a delay that will keep the tokens locked by the selector
		time.Sleep(20 * time.Second)
	}

	// Send to the ordering service and wait for finality
	_, err = context.RunView(ttx.NewOrderingAndFinalityView(tx))
	assert.NoError(err, "failed asking ordering")

	// Sanity checks:
	// - the transaction is in valid state in the vault
	vc, err = vault.Status(tx.ID())
	assert.NoError(err, "failed to retrieve vault status for transaction [%s]", tx.ID())
	assert.Equal(network.Valid, vc, "transaction [%s] should be in valid state", tx.ID())

	return tx.ID(), nil
}

type TransferWithSelectorViewFactory struct{}

func (p *TransferWithSelectorViewFactory) NewView(in []byte) (view.View, error) {
	f := &TransferWithSelectorView{Transfer: &Transfer{}}
	err := json.Unmarshal(in, f.Transfer)
	assert.NoError(err, "failed unmarshalling input")
	return f, nil
}

type PrepareTransferResult struct {
	TxID  string
	TXRaw []byte
}

// PrepareTransferView is a view that prepares a transfer transaction without broadcasting it
type PrepareTransferView struct {
	*Transfer
}

func (t *PrepareTransferView) Call(context view.Context) (interface{}, error) {
	// As a first step operation, the sender contacts the recipient's FSC node
	// to ask for the identity to use to assign ownership of the freshly created token.
	// Notice that, this step would not be required if the sender knew already which
	// identity the recipient wants to use.
	recipient, err := ttx.RequestRecipientIdentity(context, t.Recipient)
	assert.NoError(err, "failed getting recipient")

	// At this point, the sender is ready to prepare the token transaction.
	// The sender creates an anonymous transaction (this means that the resulting Fabric transaction will be signed using idemix, for example),
	// and specify the auditor that must be contacted to approve the operation.
	tx, err := ttx.NewAnonymousTransaction(
		context,
		ttx.WithAuditor(view2.GetIdentityProvider(context).Identity(t.Auditor)),
	)
	assert.NoError(err, "failed creating transaction")

	// The sender will select tokens owned by this wallet
	senderWallet := ttx.GetWallet(context, t.Wallet)
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

	// The sender is ready to collect all the required signatures.
	// In this case, the sender's and the auditor's signatures.
	// Invoke the Token Chaincode to collect endorsements on the Token Request and prepare the relative transaction.
	// This is all done in one shot running the following view.
	// Before completing, all recipients receive the approved transaction.
	// Depending on the token driver implementation, the recipient's signature might or might not be needed to make
	// the token transaction valid.
	_, err = context.RunView(ttx.NewCollectEndorsementsView(tx))
	assert.NoError(err, "failed to sign transaction")

	txRaw, err := tx.Bytes()
	assert.NoError(err, "failed to serialize transaction")

	return &PrepareTransferResult{TxID: tx.ID(), TXRaw: txRaw}, nil
}

type PrepareTransferViewFactory struct{}

func (p *PrepareTransferViewFactory) NewView(in []byte) (view.View, error) {
	f := &PrepareTransferView{Transfer: &Transfer{}}
	err := json.Unmarshal(in, f.Transfer)
	assert.NoError(err, "failed unmarshalling input")
	return f, nil
}

type BroadcastPreparedTransfer struct {
	Tx       []byte
	Finality bool
}

// BroadcastPreparedTransferView is a view that broadcasts a prepared transfer transaction
type BroadcastPreparedTransferView struct {
	*BroadcastPreparedTransfer
}

func (t *BroadcastPreparedTransferView) Call(context view.Context) (interface{}, error) {
	tx, err := ttx.NewTransactionFromBytes(context, t.Tx)
	assert.NoError(err, "failed unmarshalling transaction")

	// broadcast the transaction to the ordering service
	_, err = context.RunView(ttx.NewOrderingView(tx))
	assert.NoError(err, "failed asking ordering")

	if t.Finality {
		// wait for finality
		_, err = context.RunView(ttx.NewFinalityView(tx))
		assert.NoError(err, "failed asking ordering")
	}

	return tx.ID(), nil
}

type BroadcastPreparedTransferViewFactory struct{}

func (p *BroadcastPreparedTransferViewFactory) NewView(in []byte) (view.View, error) {
	f := &BroadcastPreparedTransferView{BroadcastPreparedTransfer: &BroadcastPreparedTransfer{}}
	err := json.Unmarshal(in, f.BroadcastPreparedTransfer)
	assert.NoError(err, "failed unmarshalling input")
	return f, nil
}

type TokenSelectorUnlock struct {
	TxID string
}

// TokenSelectorUnlockView is a view that broadcasts a prepared transfer transaction
type TokenSelectorUnlockView struct {
	*TokenSelectorUnlock
}

func (t *TokenSelectorUnlockView) Call(context view.Context) (interface{}, error) {
	assert.NoError(token2.GetManagementService(context).SelectorManager().Unlock(t.TxID), "failed to unlock tokens locked by transaction [%s]", t.TxID)

	return nil, nil
}

type TokenSelectorUnlockViewFactory struct{}

func (p *TokenSelectorUnlockViewFactory) NewView(in []byte) (view.View, error) {
	f := &TokenSelectorUnlockView{TokenSelectorUnlock: &TokenSelectorUnlock{}}
	err := json.Unmarshal(in, f.TokenSelectorUnlock)
	assert.NoError(err, "failed unmarshalling input")
	return f, nil
}

type FinalityWithTimeout struct {
	Tx      []byte
	Timeout time.Duration
}

// FinalityWithTimeoutView is a view that runs the finality view with timeout.
// The timeout is expected to happen
type FinalityWithTimeoutView struct {
	*FinalityWithTimeout
}

func (r *FinalityWithTimeoutView) Call(ctx view.Context) (interface{}, error) {
	tx, err := ttx.NewTransactionFromBytes(ctx, r.Tx)
	assert.NoError(err, "failed unmarshalling transaction")

	// broadcast the transaction to the ordering service
	start := time.Now()
	_, err = ctx.RunView(ttx.NewFinalityWithTimeoutView(tx, r.Timeout))
	end := time.Now()
	assert.Error(err)
	assert.True(strings.Contains(err.Error(), "timeout"))

	return end.Sub(start).Seconds(), nil
}

type FinalityWithTimeoutViewFactory struct{}

func (i *FinalityWithTimeoutViewFactory) NewView(in []byte) (view.View, error) {
	f := &FinalityWithTimeoutView{FinalityWithTimeout: &FinalityWithTimeout{}}
	err := json.Unmarshal(in, f.FinalityWithTimeout)
	assert.NoError(err, "failed unmarshalling input")
	return f, nil
}
