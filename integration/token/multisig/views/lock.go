/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package views

import (
	"encoding/json"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/multisig"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

// LockAction defines a policy-based lock action
type LockAction struct {
	// Amount to lock
	Amount uint64
	// Escrow is array that carries the identities of the FSC nodes of the multisig escrow
	Escrow []view.Identity
	// EscrowEIDs are the expected enrolment ids of the mutlisig escrow
	EscrowEIDs []string
}

// Lock contains the input information for a multisig-based escrow
type Lock struct {
	// Auditor is the name of the auditor that must be contacted to approve the operation
	Auditor string
	// Wallet is the identifier of the wallet that owns the tokens to transfer
	Wallet string
	// TokenIDs contains a list of token ids to transfer. If empty, tokens are selected on the spot.
	TokenIDs []*token.ID
	// Type of tokens to transfer
	Type token.Type
	// Amount to transfer
	Amount uint64
	// Escrow contains the identities of the FSC nodes of the multisig escrow
	Escrow []view.Identity
	// EscrowEIDs contains the expected enrolment ids of the mutlisig escrow
	EscrowEIDs []string
	// Retry tells if a retry must happen in case of a failure
	Retry bool
	// FailToRelease if true, it fails after transfer to trigger the Release function on the token transaction
	FailToRelease bool
	// SenderChangeRecipientData contains the recipient data that needs to be used by sender to receive the change of the transfer operation, if needed.
	// If this field is set to nil, then the token sdk generates this information as needed.
	SenderChangeRecipientData *token2.RecipientData
	// EscrowData contains the recipient data of the recipient of the transfer.
	// If nil, the view will ask the remote part to generate it, otherwise the view will notify the recipient
	// about the recipient data that will be used to the transfer.
	EscrowData []*token2.RecipientData
	// The TMS to pick in case of multiple TMSIDs
	TMSID *token2.TMSID
	// NotAnonymous true if the transaction must be anonymous, false otherwise
	NotAnonymous bool
	// Metadata contains application metadata to append to the transaction
	Metadata map[string][]byte
}

type LockView struct {
	*Lock
}

func (lv *LockView) Call(context view.Context) (txID interface{}, err error) {
	span := context.StartSpan("lock_view")
	defer span.End()
	// As a first step operation, the sender contacts the escrow FSC nodes
	// to ask for the identity to use to assign ownership of the freshly created token.
	// Notice that, this step would not be required if the sender knew already which
	// identity the escrow wants to use.

	span.AddEvent("receive_escrow_recipient_identity")
	recipients := make([]view.Identity, len(lv.Escrow))
	for k, eid := range lv.Escrow {
		recipients[k], err = ttx.RequestRecipientIdentity(context, eid, token2.WithTMSID(*lv.TMSID))
		assert.NoError(err, "failed getting recipient")
	}

	// The sender will select tokens owned by this wallet
	senderWallet := ttx.GetWallet(context, lv.Wallet, token2.WithTMSID(*lv.TMSID))
	assert.NotNil(senderWallet, "sender wallet [%s] not found", lv.Wallet)

	// At this point, the sender is ready to prepare the token transaction.
	// If NotAnonymous == fasle, then the sender creates an anonymous transaction (this means that the resulting Fabric transaction will be signed using idemix, for example),
	// and specify the auditor that must be contacted to approve the operation.
	var tx *ttx.Transaction
	txOpts := []ttx.TxOption{
		ttx.WithTMSID(*lv.TMSID),
		ttx.WithAuditor(view2.GetIdentityProvider(context).Identity(lv.Auditor)),
	}
	span.AddEvent("create_transfer")
	if !lv.NotAnonymous {
		// create an anonymous transaction (this means that the resulting Fabric transaction will be signed using idemix, for example),
		tx, err = ttx.NewAnonymousTransaction(context, txOpts...)
	} else {
		// create a nominal transaction using the default identity
		tx, err = ttx.NewTransaction(context, nil, txOpts...)
	}
	assert.NoError(err, "failed creating transaction")

	// append metadata, if any
	for k, v := range lv.Metadata {
		tx.SetApplicationMetadata(k, v)
	}

	// The sender adds a new transfer operation to the transaction following the instruction received.
	// Notice the use of `token2.WithTokenIDs(lv.TokenIDs...)`. If lv.TokenIDs is not empty, the Transfer
	// function uses those tokens, otherwise the tokens will be selected on the spot.
	// Token selection happens internally by invoking the default token selector:
	// selector, err := tx.TokenService().SelectorManager().NewSelector(tx.ID())
	// assert.NoError(err, "failed getting selector")
	// selector.Select(wallet, amount, tokenType)
	// It is also possible to pass a custom token selector to the Transfer function by using the relative opt:
	// token2.WithTokenSelector(selector).
	span.AddEvent("append_escrow_lock")
	escrowID := &multisig.MultiIdentity{
		Identities: recipients,
	}
	raw, err := escrowID.Serialize()
	assert.NoError(err, "failed serializing multi-identity")
	// This transfer sends the token to an escrow governed by a multisig
	err = tx.Transfer(
		senderWallet,
		lv.Type,
		[]uint64{lv.Amount},
		[]view.Identity{raw},
		token2.WithTokenIDs(lv.TokenIDs...),
		token2.WithRestRecipientIdentity(lv.SenderChangeRecipientData),
	)
	assert.NoError(err, "failed adding transfer action [%d:%s]", lv.Amount, view.Identity(raw).UniqueID())

	if lv.FailToRelease {
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
	/*if senderWallet.Remote() {
		// if the sender wallet is remote, then the signatures that the wallet must generate are prepared externally to this FSC node.
		// Here, we assume that the view has been called using GRPC stream
		stream := view4.GetStream(context)
		endorserOpts = append(endorserOpts, ttx.WithExternalWalletSigner(lv.Wallet, ttx.NewStreamExternalWalletSignerServer(stream)))
	}*/
	span.AddEvent("collect_endorsements")
	_, err = context.RunView(ttx.NewCollectEndorsementsView(tx, endorserOpts...))
	assert.NoError(err, "failed to sign transaction [<<<%s>>>]", tx.ID())

	// Sanity checks:
	// - the transaction is in pending state
	span.AddEvent("verify_owner")
	owner := ttx.NewOwner(context, tx.TokenService())
	vc, _, err := owner.GetStatus(tx.ID())
	assert.NoError(err, "failed to retrieve status for transaction [%s]", tx.ID())
	assert.Equal(ttx.Pending, vc, "transaction [%s] should be in busy state", tx.ID())

	// Send to the ordering service and wait for finality
	span.AddEvent("ask_ordering_finality")
	_, err = context.RunView(ttx.NewOrderingAndFinalityView(tx))
	assert.NoError(err, "failed asking ordering")

	// Sanity checks:
	// - the transaction is in confirmed state
	span.AddEvent("verify_tx_status")
	vc, _, err = owner.GetStatus(tx.ID())
	assert.NoError(err, "failed to retrieve status for transaction [%s]", tx.ID())
	assert.Equal(ttx.Confirmed, vc, "transaction [%s] should be in valid state", tx.ID())

	return tx.ID(), nil
}

type LockViewFactory struct{}

func (f *LockViewFactory) NewView(in []byte) (view.View, error) {
	v := &LockView{Lock: &Lock{}}
	err := json.Unmarshal(in, v.Lock)
	assert.NoError(err, "failed unmarshalling input")
	return v, nil
}

type LockWithSelectorView struct {
	*Lock
}

func (lv *LockWithSelectorView) Call(context view.Context) (interface{}, error) {
	span := context.StartSpan("lock_selector_iew")
	defer span.End()
	// As a first step operation, the sender contacts the escrow FSC nodes
	// to ask for the identity to use to assign ownership of the freshly created token.
	// Notice that, this step would not be required if the sender knew already which
	// identity the escrow wants to use.

	span.AddEvent("receive_escrow_recipient_identity")
	recipients := make([]view.Identity, len(lv.Escrow))
	var err error
	for k, eid := range lv.Escrow {
		recipients[k], err = ttx.RequestRecipientIdentity(context, eid, token2.WithTMSID(*lv.TMSID))
		assert.NoError(err, "failed getting recipient %s", eid)
	}

	// At this point, the sender is ready to prepare the token transaction.
	// The sender creates an anonymous transaction (this means that the resulting Fabric transaction will be signed using idemix, for example),
	// and specify the auditor that must be contacted to approve the operation.
	tx, err := ttx.NewAnonymousTransaction(
		context,
		ttx.WithAuditor(view2.GetIdentityProvider(context).Identity(lv.Auditor)),
	)
	assert.NoError(err, "failed creating transaction")

	// The sender will select tokens owned by this wallet
	senderWallet := ttx.GetWallet(context, lv.Wallet)
	assert.NotNil(senderWallet, "sender wallet [%s] not found", lv.Wallet)

	// If no specific tokens are requested, then a custom token selection process start
	precision := token2.GetManagementService(context).PublicParametersManager().PublicParameters().Precision()
	amount, err := token.UInt64ToQuantity(lv.Amount, precision)
	assert.NoError(err, "failed to convert to quantity")

	if len(lv.TokenIDs) == 0 {
		// The sender uses the default token selector each transaction comes equipped with
		selector, err := tx.Selector()
		defer func() {
			if err := tx.CloseSelector(); err != nil {
				logger.Errorf("failed closing selector [%s]", err)
			}
		}()
		assert.NoError(err, "failed getting token selector")

		// The sender tries to select the requested amount of tokens of the passed type.
		// If a failure happens, the sender retries up to 5 times, waiting 10 seconds after each failure.
		// This is just an example, any other policy can be implemented.
		var ids []*token.ID
		var sum token.Quantity

		for i := 0; i < 5; i++ {
			// Select the request amount of tokens of the given type
			ids, sum, err = selector.Select(
				ttx.GetWallet(context, lv.Wallet),
				amount.Decimal(),
				lv.Type,
			)
			// If an error occurs and retry has been asked, then wait first a bit
			if err != nil && lv.Retry {
				time.Sleep(10 * time.Second)
				continue
			}
			break
		}
		if err != nil {
			// If finally not enough tokens were available, the sender can check what was the cause of the error:
			switch {
			case errors.HasCause(err, token2.SelectorInsufficientFunds):
				assert.NoError(err, "pineapple")
			case errors.HasCause(err, token2.SelectorSufficientButLockedFunds):
				assert.NoError(err, "lemonade")
			case errors.HasCause(err, token2.SelectorSufficientButNotCertifiedFunds):
				assert.NoError(err, "mandarin")
			case errors.HasCause(err, token2.SelectorSufficientFundsButConcurrencyIssue):
				assert.NoError(err, "peach")
			default:
				assert.NoError(err, "system failure")
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
			assert.Equal(lv.Type, tok.Type, "expected token of type [%s], got [%s]", lv.Type, tok.Type)
			// Add the quantity to the total
			q, err := token.ToQuantity(tok.Quantity, precision)
			assert.NoError(err, "failed converting quantity")
			recomputedSum = recomputedSum.Add(q)
		}
		// Is the recomputed sum correct?
		assert.True(sum.Cmp(recomputedSum) == 0, "sums do not match")
		// Is the amount selected equal or larger than what requested?
		assert.False(sum.Cmp(amount) < 0, "if this point is reached, funds are sufficient")

		lv.TokenIDs = ids
	}

	// The sender adds a new transfer operation to the transaction following the instruction received.
	// Notice the use of `token2.WithTokenIDs(lv.TokenIDs...)` to pass the token ids selected above.
	span.AddEvent("append_escrow_lock")
	escrowID := &multisig.MultiIdentity{
		Identities: recipients,
	}
	raw, err := escrowID.Serialize()
	assert.NoError(err, "failed serializing multi-identity")
	// This transfer sends the token to an escrow governed by a multisig
	err = tx.Transfer(
		ttx.GetWallet(context, lv.Wallet),
		lv.Type,
		[]uint64{lv.Amount},
		[]view.Identity{raw},
		token2.WithTokenIDs(lv.TokenIDs...),
	)

	assert.NoError(err, "failed transferring tokens")

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
	// - the transaction is in pending state
	owner := ttx.NewOwner(context, tx.TokenService())
	vc, _, err := owner.GetStatus(tx.ID())
	assert.NoError(err, "failed to retrieve status for transaction [%s]", tx.ID())
	assert.Equal(ttx.Pending, vc, "transaction [%s] should be in busy state", tx.ID())

	if !lv.Retry {
		// Introduce a delay that will keep the tokens locked by the selector
		time.Sleep(20 * time.Second)
	}

	// Send to the ordering service and wait for finality
	_, err = context.RunView(ttx.NewOrderingAndFinalityView(tx))
	assert.NoError(err, "failed asking ordering")

	// Sanity checks:
	// - the transaction is in confirmed state
	vc, _, err = owner.GetStatus(tx.ID())
	assert.NoError(err, "failed to retrieve status for transaction [%s]", tx.ID())
	assert.Equal(ttx.Confirmed, vc, "transaction [%s] should be in valid state", tx.ID())

	return tx.ID(), nil
}

type LockWithSelectorViewFactory struct{}

func (f *LockWithSelectorViewFactory) NewView(in []byte) (view.View, error) {
	v := &LockWithSelectorView{Lock: &Lock{}}
	err := json.Unmarshal(in, v.Lock)
	assert.NoError(err, "failed unmarshalling input")
	return v, nil
}
