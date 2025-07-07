/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package htlc

import (
	"encoding/json"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/id"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/htlc"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

// FastExchange contains the input information to fast exchange tokens
type FastExchange struct {
	// Recipient is the identity of the recipient's FSC node
	Recipient view.Identity
	// TMSID identifies the TMS to use to perform the token operation
	TMSID1 token.TMSID
	// Type of tokens to transfer
	Type1 token2.Type
	// Amount to transfer
	Amount1 uint64

	TMSID2  token.TMSID
	Type2   token2.Type
	Amount2 uint64

	// ReclamationDeadline is the time after which we can reclaim the funds in case they were not transferred
	ReclamationDeadline time.Duration
}

// FastExchangeInitiatorView is the view of a party who wants to perform an exchange with a single view
type FastExchangeInitiatorView struct {
	*FastExchange
}

func (v *FastExchangeInitiatorView) Call(context view.Context) (interface{}, error) {
	// Preliminary:
	// 1. Exchange recipient identities
	// 2. Agree on the terms
	// 3. Lock assets
	// 4. Initiator claims
	// 5. Responder retrieves pre-images and claims

	// We assume that the initiator and the responder are on both networks
	me1, recipient, err := htlc.ExchangeRecipientIdentities(context, "", v.Recipient, token.WithTMSID(v.TMSID1))
	assert.NoError(err, "failed getting recipient identity of participants in the first network")
	me2, sender, err := htlc.ExchangeRecipientIdentities(context, "", v.Recipient, token.WithTMSID(v.TMSID2))
	assert.NoError(err, "failed getting recipient identity of participants in the second network")

	_, err = context.RunView(htlc.NewDistributeTermsView(recipient, &htlc.Terms{
		ReclamationDeadline: v.ReclamationDeadline,
		TMSID1:              v.TMSID1,
		Type1:               v.Type1,
		Amount1:             v.Amount1,
		TMSID2:              v.TMSID2,
		Type2:               v.Type2,
		Amount2:             v.Amount2,
	}))
	assert.NoError(err, "failed to distribute terms")

	// Initiator's Leg
	var preImage []byte
	idProvider, err := id.GetProvider(context)
	assert.NoError(err, "failed getting id provider")
	_, err = view2.RunCall(context, func(context view.Context) (interface{}, error) {
		tx, err := htlc.NewAnonymousTransaction(
			context,
			ttx.WithAuditor(idProvider.Identity("auditor")),
			ttx.WithTMSID(v.TMSID1),
		)
		assert.NoError(err, "failed to create an htlc transaction")

		wallet := htlc.GetWallet(context, "", token.WithTMSID(v.TMSID1))
		assert.NotNil(wallet, "wallet not found")

		preImage, err = tx.Lock(
			context.Context(),
			wallet,
			me1,
			v.Type1,
			v.Amount1,
			recipient,
			v.ReclamationDeadline,
			htlc.WithHash(nil),
		)
		assert.NoError(err, "failed adding a lock action")

		_, err = context.RunView(htlc.NewCollectEndorsementsView(tx))
		assert.NoError(err, "failed to collect endorsements for htlc transaction")

		_, err = context.RunView(htlc.NewOrderingAndFinalityView(tx))
		assert.NoError(err, "failed to commit htlc transaction")

		return nil, nil
	})
	assert.NoError(err, "failed completing cycle initiator's leg")

	// Responder's Leg (here the initiator plays the role of responder)
	session, err := context.GetSession(context.Initiator(), v.Recipient)
	assert.NoError(err, "failed to get the session")
	_, err = view2.AsResponder(context, session,
		func(context view.Context) (interface{}, error) {
			tx, err := htlc.ReceiveTransaction(context)
			assert.NoError(err, "failed to receive tokens")

			outputs, err := tx.Outputs()
			assert.NoError(err, "failed getting outputs")
			assert.True(outputs.Count() >= 1, "expected at least one output, got [%d]", outputs.Count())
			outputs = outputs.ByScript()
			assert.True(outputs.Count() == 1, "expected only one htlc output, got [%d]", outputs.Count())
			script := outputs.ScriptAt(0)
			assert.NotNil(script, "expected an htlc script")
			assert.True(me2.Equal(script.Recipient), "expected me as recipient of the script")
			assert.True(sender.Equal(script.Sender), "expected recipient as sender of the script")

			_, err = context.RunView(htlc.NewAcceptView(tx))
			assert.NoError(err, "failed to accept new tokens")

			_, err = context.RunView(htlc.NewFinalityView(tx))
			assert.NoError(err, "new tokens were not committed")

			return nil, nil
		},
	)
	assert.NoError(err, "failed to complete responder's leg (as responder)")

	// The initiator claims the responder's script, this can be done in a fresh context
	_, err = view2.Initiate(context, &ClaimView{
		&Claim{
			TMSID:    v.TMSID2,
			Wallet:   "",
			PreImage: preImage,
		},
	})
	assert.NoError(err, "failed to complete responder's leg (as initiator)")

	return nil, nil
}

type FastExchangeInitiatorViewFactory struct{}

func (f *FastExchangeInitiatorViewFactory) NewView(in []byte) (view.View, error) {
	v := &FastExchangeInitiatorView{FastExchange: &FastExchange{}}
	err := json.Unmarshal(in, v.FastExchange)
	assert.NoError(err, "failed unmarshalling input")

	return v, nil
}

type FastExchangeResponderView struct{}

func (v *FastExchangeResponderView) Call(context view.Context) (interface{}, error) {
	// Preliminary:
	// 1. Exchange recipient identities
	// 2. Agree on the terms
	me1, sender, err := htlc.RespondExchangeRecipientIdentities(context)
	assert.NoError(err, "failed to respond to identity request in the first network")

	me2, recipient, err := htlc.RespondExchangeRecipientIdentities(context)
	assert.NoError(err, "failed to respond to identity request in the second network")

	terms, err := htlc.ReceiveTerms(context)
	assert.NoError(err, "failed to receive the terms")

	assert.NoError(terms.Validate(), "failed to validate the terms")

	// Respond to Initiator's Leg
	var script *htlc.Script
	{
		tx, err := htlc.ReceiveTransaction(context)
		assert.NoError(err, "failed to receive tokens")

		outputs, err := tx.Outputs()
		assert.NoError(err, "failed getting outputs")
		assert.True(outputs.Count() >= 1, "expected at least one output, got [%d]", outputs.Count())
		outputs = outputs.ByScript()
		assert.True(outputs.Count() == 1, "expected only one htlc output, got [%d]", outputs.Count())
		script = outputs.ScriptAt(0)
		assert.NotNil(script, "expected an htlc script")
		assert.True(me1.Equal(script.Recipient), "expected me as recipient of the script")
		assert.True(sender.Equal(script.Sender), "expected sender as sender of the script")

		_, err = context.RunView(htlc.NewAcceptView(tx))
		assert.NoError(err, "failed to accept new tokens")

		_, err = context.RunView(htlc.NewFinalityView(tx))
		assert.NoError(err, "new tokens were not committed")
	}

	// Initiate Responder's Leg
	idProvider, err := id.GetProvider(context)
	assert.NoError(err, "failed getting id provider")
	_, err = view2.AsInitiatorCall(context, v, func(context view.Context) (interface{}, error) {
		tx, err := htlc.NewAnonymousTransaction(
			context,
			ttx.WithAuditor(idProvider.Identity("auditor")),
			ttx.WithTMSID(terms.TMSID2),
		)
		assert.NoError(err, "failed to create an htlc transaction")

		wallet := htlc.GetWallet(context, "", token.WithTMSID(terms.TMSID2))
		assert.NotNil(wallet, "wallet not found")

		_, err = tx.Lock(
			context.Context(),
			wallet,
			me2,
			terms.Type2,
			terms.Amount2,
			recipient,
			terms.ReclamationDeadline, // TODO maybe use a different deadline
			htlc.WithHash(script.HashInfo.Hash),
		)
		assert.NoError(err, "failed adding a lock action")

		_, err = context.RunView(htlc.NewCollectEndorsementsView(tx))
		assert.NoError(err, "failed to collect endorsements on htlc transaction")

		_, err = context.RunView(htlc.NewOrderingAndFinalityView(tx))
		assert.NoError(err, "failed to commit htlc transaction")

		return nil, nil
	})
	assert.NoError(err, "failed completing responder's leg (as initiator)")

	time.Sleep(30 * time.Second)

	// Claim initiator's script, we don't need any interaction with the initiator (FastExchangeInitiatorView)
	_, err = view2.Initiate(context, &ClaimView{
		&Claim{
			TMSID:       terms.TMSID1,
			Wallet:      "",
			Script:      script,
			ScriptTMSID: terms.TMSID2,
		},
	})
	assert.NoError(err, "failed completing responder's leg (as initiator)")

	return nil, nil
}
