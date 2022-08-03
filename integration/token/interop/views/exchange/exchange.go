/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package exchange

import (
	"encoding/json"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	view3 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/exchange"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
)

// FastExchange contains the input information to fast exchange a token
type FastExchange struct {
	// Recipient is the identity of the recipient's FSC node
	Recipient view.Identity
	// TMSID identifies the TMS to use to perform the token operation
	TMSID1 token.TMSID
	// Type of tokens to transfer
	Type1 string
	// Amount to transfer
	Amount1 uint64

	TMSID2  token.TMSID
	Type2   string
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
	me1, recipient, err := exchange.ExchangeRecipientIdentities(context, "", v.Recipient, token.WithTMSID(v.TMSID1))
	assert.NoError(err, "failed getting recipient identity of participants in the first network")
	me2, sender, err := exchange.ExchangeRecipientIdentities(context, "", v.Recipient, token.WithTMSID(v.TMSID2))
	assert.NoError(err, "failed getting recipient identity of participants in the second network")

	_, err = context.RunView(exchange.NewDistributeTermsView(recipient, &exchange.Terms{
		ReclamationDeadline: v.ReclamationDeadline,
		TMSID1:              v.TMSID1,
		Type1:               v.Type1,
		Amount1:             v.Amount1,
		TMSID2:              v.TMSID2,
		Type2:               v.Type2,
		Amount2:             v.Amount2,
	}))
	assert.NoError(err, "failed to distribute terms")

	// Initiator's Leg (HTLC)
	var preImage []byte
	_, err = view2.RunCall(context, func(context view.Context) (interface{}, error) {
		tx, err := exchange.NewTransaction(
			context,
			fabric.GetIdentityProvider(context, v.TMSID1.Network).DefaultIdentity(),
			ttx.WithAuditor(view3.GetIdentityProvider(context).Identity("auditor")),
			ttx.WithTMSID(v.TMSID1),
		)
		assert.NoError(err, "failed to create an exchange transaction")

		wallet := exchange.GetWallet(context, "", token.WithTMSID(v.TMSID1))
		assert.NotNil(wallet, "wallet not found")

		preImage, err = tx.Exchange(
			wallet,
			me1,
			v.Type1,
			v.Amount1,
			recipient,
			v.ReclamationDeadline,
			exchange.WithHash(nil),
		)
		assert.NoError(err, "failed adding exchange")

		_, err = context.RunView(exchange.NewCollectEndorsementsView(tx))
		assert.NoError(err, "failed to collect endorsements for exchange transaction")

		_, err = context.RunView(exchange.NewOrderingAndFinalityView(tx))
		assert.NoError(err, "failed to commit exchange transaction")

		return nil, nil
	})
	assert.NoError(err, "failed completing cycle initiator's leg")

	// Responder's Leg (here the initiator plays the role of responder) (HTLC)
	session, err := context.GetSession(context.Initiator(), v.Recipient)
	assert.NoError(err, "failed to get the session")
	_, err = view2.AsResponder(context, session,
		func(context view.Context) (interface{}, error) {
			tx, err := exchange.ReceiveTransaction(context)
			assert.NoError(err, "failed to receive tokens")

			outputs, err := tx.Outputs()
			assert.NoError(err, "failed getting outputs")
			assert.True(outputs.Count() >= 1, "expected at least one output, got [%d]", outputs.Count())
			outputs = outputs.ByScript()
			assert.True(outputs.Count() == 1, "expected only one exchange output, got [%d]", outputs.Count())
			script := outputs.ScriptAt(0)
			assert.NotNil(script, "expected an exchange script")
			assert.True(me2.Equal(script.Recipient), "expected me as recipient of the script")
			assert.True(sender.Equal(script.Sender), "expected recipient as sender of the script")

			_, err = context.RunView(exchange.NewAcceptView(tx))
			assert.NoError(err, "failed to accept new tokens")

			_, err = context.RunView(exchange.NewFinalityView(tx))
			assert.NoError(err, "new tokens were not committed")

			return nil, nil
		},
	)
	assert.NoError(err, "failed to complete responder's leg (as responder)")

	// The initiator claims the responder's script
	_, err = view2.RunCall(context, func(context view.Context) (interface{}, error) {
		wallet := exchange.GetWallet(context, "", token.WithTMSID(v.TMSID2))
		assert.NotNil(wallet, "wallet not found")

		matched, err := exchange.Wallet(context, wallet, token.WithTMSID(v.TMSID2)).ListByPreImage(preImage)
		assert.NoError(err, "cannot retrieve list of expired exchange")
		assert.True(len(matched) == 1, "expected only one exchange script to match, got [%d]", len(matched))

		tx, err := exchange.NewTransaction(
			context,
			fabric.GetIdentityProvider(context, v.TMSID2.Network).DefaultIdentity(),
			ttx.WithAuditor(view3.GetIdentityProvider(context).Identity("auditor")),
			ttx.WithTMSID(v.TMSID2),
		)
		assert.NoError(err, "failed to create an exchange transaction")
		assert.NoError(tx.Claim(wallet, matched[0], preImage), "failed adding a claim for [%s]", matched[0].Id)

		_, err = context.RunView(exchange.NewCollectEndorsementsView(tx))
		assert.NoError(err, "failed to collect endorsements for exchange transaction")

		_, err = context.RunView(exchange.NewOrderingAndFinalityView(tx))
		assert.NoError(err, "failed to commit exchange transaction")

		return nil, nil
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
	me1, sender, err := exchange.RespondExchangeRecipientIdentities(context)
	assert.NoError(err, "failed to respond to identity request in the first network")

	me2, recipient, err := exchange.RespondExchangeRecipientIdentities(context)
	assert.NoError(err, "failed to respond to identity request in the second network")

	terms, err := exchange.ReceiveTerms(context)
	assert.NoError(err, "failed to receive the terms")

	// TODO: validate the terms and tell the initiator if they are accepted

	// Initiator's Leg (HTLC)
	var script *exchange.Script
	_, err = view2.AsInitiatorCall(context, v, func(context view.Context) (interface{}, error) {
		tx, err := exchange.ReceiveTransaction(context)
		assert.NoError(err, "failed to receive tokens")

		outputs, err := tx.Outputs()
		assert.NoError(err, "failed getting outputs")
		assert.True(outputs.Count() >= 1, "expected at least one output, got [%d]", outputs.Count())
		outputs = outputs.ByScript()
		assert.True(outputs.Count() == 1, "expected only one exchange output, got [%d]", outputs.Count())
		script = outputs.ScriptAt(0)
		assert.NotNil(script, "expected an exchange script")
		assert.True(me1.Equal(script.Recipient), "expected me as recipient of the script")
		assert.True(sender.Equal(script.Sender), "expected sender as sender of the script")

		_, err = context.RunView(exchange.NewAcceptView(tx))
		assert.NoError(err, "failed to accept new tokens")

		_, err = context.RunView(exchange.NewFinalityView(tx))
		assert.NoError(err, "new tokens were not committed")

		return nil, nil
	})
	assert.NoError(err, "failed completing initiator's leg")

	// Responder's Leg (HTLC)
	_, err = view2.AsInitiatorCall(context, v, func(context view.Context) (interface{}, error) {
		tx, err := exchange.NewTransaction(
			context,
			fabric.GetIdentityProvider(context, terms.TMSID2.Network).DefaultIdentity(),
			ttx.WithAuditor(view3.GetIdentityProvider(context).Identity("auditor")),
			ttx.WithTMSID(terms.TMSID2),
		)
		assert.NoError(err, "failed to create an exchange transaction")

		wallet := exchange.GetWallet(context, "", token.WithTMSID(terms.TMSID2))
		assert.NotNil(wallet, "wallet not found")

		_, err = tx.Exchange(
			wallet,
			me2,
			terms.Type2,
			terms.Amount2,
			recipient,
			terms.ReclamationDeadline, // TODO maybe use a different deadline
			exchange.WithHash(script.HashInfo.Hash),
		)
		assert.NoError(err, "failed adding exchange")

		_, err = context.RunView(exchange.NewCollectEndorsementsView(tx))
		assert.NoError(err, "failed to collect endorsements on exchange transaction")

		_, err = context.RunView(exchange.NewOrderingAndFinalityView(tx))
		assert.NoError(err, "failed to commit exchange transaction")

		return nil, nil
	})
	assert.NoError(err, "failed completing responder's leg (as initiator)")

	// Claim initiator's script
	_, err = view2.AsInitiatorCall(context, v, func(context view.Context) (interface{}, error) {
		// Scan for the pre-image, this will be the signal that the initiator has claimed responder's script
		// TODO: fix timeout
		preImage, err := exchange.ScanForPreImage(context, script.HashInfo.Hash, script.HashInfo.HashFunc, script.HashInfo.HashEncoding, 5*time.Minute, token.WithTMSID(terms.TMSID2))
		// if an error occurred, reclaim
		if err != nil {
			// reclaim
			assert.NoError(err, "failed to receive the preImage")
		}

		// claim initiator's script
		wallet := exchange.GetWallet(context, "", token.WithTMSID(terms.TMSID1))
		assert.NotNil(wallet, "wallet not found")
		matched, err := exchange.Wallet(context, wallet, token.WithTMSID(terms.TMSID1)).ListByPreImage(preImage)
		assert.NoError(err, "cannot retrieve list of expired exchange")
		assert.True(len(matched) == 1, "expected only one exchange script to match, got [%d]", len(matched))

		tx, err := exchange.NewTransaction(
			context,
			fabric.GetIdentityProvider(context, terms.TMSID1.Network).DefaultIdentity(),
			ttx.WithAuditor(view3.GetIdentityProvider(context).Identity("auditor")),
			ttx.WithTMSID(terms.TMSID1),
		)
		assert.NoError(err, "failed to create an exchange transaction")
		assert.NoError(tx.Claim(wallet, matched[0], preImage), "failed adding a claim for [%s]", matched[0].Id)

		_, err = context.RunView(exchange.NewCollectEndorsementsView(tx))
		assert.NoError(err, "failed to collect endorsements for exchange transaction")

		_, err = context.RunView(exchange.NewOrderingAndFinalityView(tx))
		assert.NoError(err, "failed to commit exchange transaction")

		return nil, nil
	})
	assert.NoError(err, "failed completing responder's leg (as initiator)")

	return nil, nil
}
