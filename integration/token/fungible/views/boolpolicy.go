/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package views

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/id"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	bptx "github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx/boolpolicy"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

// PolicyLock contains the input parameters for locking tokens under a policy identity.
type PolicyLock struct {
	// Auditor is the name of the auditor FSC node.
	Auditor string
	// Wallet is the sender's owner wallet identifier.
	Wallet string
	// Type is the token type to transfer.
	Type token.Type
	// Amount is the number of tokens to lock.
	Amount uint64
	// Policy is the boolean policy expression, e.g. "$0 OR $1".
	Policy string
	// PolicyParties are the FSC node identities that become co-owners.
	PolicyParties []view.Identity
	// TMSID optionally pins a specific TMS.
	TMSID *token2.TMSID
	// NotAnonymous disables anonymous transactions when true.
	NotAnonymous bool
}

// PolicyLockView transfers tokens to a policy identity owner.
type PolicyLockView struct {
	*PolicyLock
}

// Call implements view.View.
func (lv *PolicyLockView) Call(context view.Context) (interface{}, error) {
	// Collect a policy identity from all co-owner parties.
	recipient, err := bptx.RequestRecipientIdentity(context, lv.Policy, lv.PolicyParties, token2.WithTMSIDPointer(lv.TMSID))
	assert.NoError(err, "failed requesting policy recipient identity")

	senderWallet := ttx.GetWallet(context, lv.Wallet, token2.WithTMSIDPointer(lv.TMSID))
	assert.NotNil(senderWallet, "sender wallet [%s] not found", lv.Wallet)

	idProvider, err := id.GetProvider(context)
	assert.NoError(err, "failed getting id provider")
	tx, err := ttx.NewTransaction(context, nil,
		ttx.WithTMSIDPointer(lv.TMSID),
		ttx.WithAuditor(idProvider.Identity(lv.Auditor)),
		ttx.WithAnonymousTransaction(!lv.NotAnonymous),
	)
	assert.NoError(err, "failed creating transaction")

	assert.NoError(bptx.Wrap(tx).Lock(senderWallet, lv.Type, lv.Amount, recipient), "failed adding lock")

	_, err = context.RunView(ttx.NewCollectEndorsementsView(tx))
	assert.NoError(err, "failed to collect endorsements")

	owner := ttx.NewOwner(context, tx.TokenService())
	vc, _, err := owner.GetStatus(context.Context(), tx.ID())
	assert.NoError(err, "failed to retrieve status for transaction [%s]", tx.ID())
	assert.Equal(ttx.Pending, vc, "transaction [%s] should be in pending state", tx.ID())

	_, err = context.RunView(ttx.NewOrderingAndFinalityView(tx))
	assert.NoError(err, "failed asking ordering")

	vc, _, err = owner.GetStatus(context.Context(), tx.ID())
	assert.NoError(err, "failed to retrieve status for transaction [%s]", tx.ID())
	assert.Equal(ttx.Confirmed, vc, "transaction [%s] should be in confirmed state", tx.ID())

	return tx.ID(), nil
}

// PolicyLockViewFactory creates PolicyLockView instances from serialised parameters.
type PolicyLockViewFactory struct{}

// NewView implements view.Factory.
func (f *PolicyLockViewFactory) NewView(in []byte) (view.View, error) {
	v := &PolicyLockView{PolicyLock: &PolicyLock{}}
	assert.NoError(json.Unmarshal(in, v.PolicyLock), "failed unmarshalling input")

	return v, nil
}

// ---------------------------------------------------------------------------
// OR-policy spend: a single co-owner spends unilaterally.
// ---------------------------------------------------------------------------

// PolicySpendOR contains the parameters for spending a policy token when the policy is OR.
type PolicySpendOR struct {
	// Auditor is the name of the auditor FSC node.
	Auditor string
	// Wallet is the spender's owner wallet identifier.
	Wallet string
	// Recipient is the FSC node identity that will receive the tokens.
	Recipient view.Identity
	// TMSID optionally pins a specific TMS.
	TMSID *token2.TMSID
	// TokenType is the type of tokens to spend.
	TokenType token.Type
}

// PolicySpendORView spends a policy token whose OR policy can be satisfied by a single signer.
type PolicySpendORView struct {
	*PolicySpendOR
}

// Call implements view.View.
func (r *PolicySpendORView) Call(context view.Context) (interface{}, error) {
	serviceOpts := ServiceOpts(r.TMSID)
	recipient, err := ttx.RequestRecipientIdentity(context, r.Recipient, serviceOpts...)
	assert.NoError(err, "failed getting recipient")

	spendWallet := bptx.GetWallet(context, r.Wallet, serviceOpts...)
	assert.NotNil(spendWallet, "wallet [%s] not found", r.Wallet)

	pWallet := bptx.Wallet(context, spendWallet)
	assert.NotNil(pWallet, "policy wallet wrapper for [%s] not found", r.Wallet)

	matched, err := pWallet.ListTokens(context.Context(), token2.WithType(r.TokenType))
	assert.NoError(err, "failed to list policy tokens")
	assert.True(matched.Count() == 1, "expected exactly one policy token, got [%d]", matched.Count())

	idProvider, err := id.GetProvider(context)
	assert.NoError(err, "failed getting id provider")
	tx, err := ttx.NewAnonymousTransaction(
		context,
		TxOpts(r.TMSID, ttx.WithAuditor(idProvider.Identity(r.Auditor)))...,
	)
	assert.NoError(err, "failed to create policy transaction")
	assert.NoError(bptx.Wrap(tx).Spend(spendWallet, matched.At(0), recipient), "failed adding spend")

	_, err = context.RunView(ttx.NewCollectEndorsementsView(tx))
	assert.NoError(err, "failed to collect endorsements on policy OR spend")

	_, err = context.RunView(ttx.NewOrderingAndFinalityView(tx))
	assert.NoError(err, "failed to commit policy OR spend")

	return tx.ID(), nil
}

// PolicySpendORViewFactory creates PolicySpendORView instances.
type PolicySpendORViewFactory struct{}

// NewView implements view.Factory.
func (f *PolicySpendORViewFactory) NewView(in []byte) (view.View, error) {
	v := &PolicySpendORView{PolicySpendOR: &PolicySpendOR{}}
	assert.NoError(json.Unmarshal(in, v.PolicySpendOR), "failed unmarshalling input")

	return v, nil
}

// ---------------------------------------------------------------------------
// AND-policy spend: all co-owners must endorse.
// ---------------------------------------------------------------------------

// PolicySpendAND contains parameters for an AND-policy spend where all parties must sign.
type PolicySpendAND struct {
	// Auditor is the name of the auditor FSC node.
	Auditor string
	// Wallet is the initiating spender's owner wallet identifier.
	Wallet string
	// Recipient is the FSC node identity that will receive the tokens.
	Recipient view.Identity
	// TMSID optionally pins a specific TMS.
	TMSID *token2.TMSID
	// TokenType is the type of tokens to spend.
	TokenType token.Type
}

// PolicySpendANDView initiates spending a policy-AND token: it notifies co-owners, then
// collects all endorsements before committing.
type PolicySpendANDView struct {
	*PolicySpendAND
}

// Call implements view.View.
func (r *PolicySpendANDView) Call(context view.Context) (interface{}, error) {
	serviceOpts := ServiceOpts(r.TMSID)
	recipient, err := ttx.RequestRecipientIdentity(context, r.Recipient, serviceOpts...)
	assert.NoError(err, "failed getting recipient")

	spendWallet := bptx.GetWallet(context, r.Wallet, serviceOpts...)
	assert.NotNil(spendWallet, "wallet [%s] not found", r.Wallet)

	pWallet := bptx.Wallet(context, spendWallet)
	assert.NotNil(pWallet, "policy wallet wrapper for [%s] not found", r.Wallet)

	matched, err := pWallet.ListTokens(context.Context(), token2.WithType(r.TokenType))
	assert.NoError(err, "failed to list policy tokens")
	assert.True(matched.Count() == 1, "expected exactly one policy token, got [%d]", matched.Count())

	// Notify co-owners about the intent to spend.
	_, err = context.RunView(bptx.NewRequestSpendView(matched.At(0), serviceOpts...))
	assert.NoError(err, "failed to request policy spend")

	idProvider, err := id.GetProvider(context)
	assert.NoError(err, "failed getting id provider")
	tx, err := ttx.NewAnonymousTransaction(
		context,
		TxOpts(r.TMSID, ttx.WithAuditor(idProvider.Identity(r.Auditor)))...,
	)
	assert.NoError(err, "failed to create policy AND transaction")
	assert.NoError(bptx.Wrap(tx).Spend(spendWallet, matched.At(0), recipient), "failed adding spend")

	_, err = context.RunView(ttx.NewCollectEndorsementsView(tx))
	assert.NoError(err, "failed to collect endorsements on policy AND spend")

	_, err = context.RunView(ttx.NewOrderingAndFinalityView(tx))
	assert.NoError(err, "failed to commit policy AND spend")

	return tx.ID(), nil
}

// PolicySpendANDViewFactory creates PolicySpendANDView instances.
type PolicySpendANDViewFactory struct{}

// NewView implements view.Factory.
func (f *PolicySpendANDViewFactory) NewView(in []byte) (view.View, error) {
	v := &PolicySpendANDView{PolicySpendAND: &PolicySpendAND{}}
	assert.NoError(json.Unmarshal(in, v.PolicySpendAND), "failed unmarshalling input")

	return v, nil
}

// PolicyAcceptSpendView is the co-owner responder for AND-policy spend requests.
// It mirrors MultiSigAcceptSpendView.
type PolicyAcceptSpendView struct{}

// Call implements view.View.
func (m *PolicyAcceptSpendView) Call(context view.Context) (interface{}, error) {
	request, err := bptx.ReceiveSpendRequest(context)
	assert.NoError(err, "failed receiving policy spend request")
	assert.NotNil(request.Token, "request doesn't contain a token")

	tx, err := bptx.EndorseSpend(context, request)
	assert.NoError(err, "failed approving policy spend")

	owner := ttx.NewOwner(context, tx.TokenService())
	vc, _, err := owner.GetStatus(context.Context(), tx.ID())
	assert.NoError(err, "failed to retrieve status for transaction [%s]", tx.ID())
	assert.Equal(ttx.Pending, vc, "transaction [%s] should be in pending state", tx.ID())

	_, err = context.RunView(bptx.NewFinalityView(tx))
	assert.NoError(err, "new tokens were not committed")

	vc, _, err = owner.GetStatus(context.Context(), tx.ID())
	assert.NoError(err, "failed to retrieve status for transaction [%s]", tx.ID())
	assert.Equal(ttx.Confirmed, vc, "transaction [%s] should be in confirmed state", tx.ID())

	return nil, nil
}
