/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package views

import (
	"encoding/json"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type Conversion struct {
	// TMSID the token management service identifier
	TMSID token.TMSID
	// Wallet of the recipient of the cash to be issued
	Wallet string
	// Amount represent the number of units of a certain token type stored in the token
	Amount uint64
	// TokenType is the type of token to issue
	TokenType token2.Type
	// Issuer identifies the issuer
	Issuer string
	// Recipient information
	RecipientData *token.RecipientData
	// NotAnonymous true if the transaction must be anonymous, false otherwise
	NotAnonymous bool
}

type ConversionInitiatorView struct {
	*Conversion
}

func (i *ConversionInitiatorView) Call(context view.Context) (interface{}, error) {
	span := context.StartSpan("conversion_initiator_view")
	defer span.End()

	// First, the initiator selects the tokens that are not spendable
	w := token.GetManagementService(context, token.WithTMSID(i.TMSID)).WalletManager().OwnerWallet(i.Wallet)
	assert.NotNil(w, "cannot find wallet [%s:%s]", i.TMSID, i.Wallet)

	tokensProvider, err := tokens.GetProvider(context)
	assert.NoError(err, "failed getting tokens provider")
	tokens, err := tokensProvider.Tokens(i.TMSID)
	assert.NoError(err, "failed getting tokens")
	assert.NotNil(tokens, "failed getting tokens")
	it, err := tokens.UnspendableTokensIteratorBy(context.Context(), w.ID(), i.TokenType)
	assert.NoError(err, "failed getting tokens iterator")
	unspendableTokens, err := collections.ReadAll(it)
	assert.NoError(err, "failed getting tokens")

	// Then, the initiator sends a conversion request to the issuer.
	// If the initiator has already some recipient data, it uses that directly
	var id view.Identity
	var session view.Session
	if i.RecipientData != nil {
		// Use the passed RecipientData.
		// First register it locally
		assert.NoError(w.RegisterRecipient(i.RecipientData), "failed to register remote recipient")
		// Then request conversion
		span.AddEvent("request_conversion_for_recipient")
		id, session, err = ttx.RequestConversionForRecipient(
			context,
			view.Identity(i.Issuer),
			i.Wallet,
			unspendableTokens,
			i.NotAnonymous,
			i.RecipientData,
			token.WithTMSID(i.TMSID),
		)
	} else {
		span.AddEvent("request_conversion")
		id, session, err = ttx.RequestConversion(
			context,
			view.Identity(i.Issuer),
			i.Wallet,
			unspendableTokens,
			i.NotAnonymous,
			token.WithTMSID(i.TMSID),
		)
	}
	assert.NoError(err, "failed to send conversion request")

	// Request conversion

	// At this point we have an inversion of roles.
	// The initiator becomes a responder.
	// This is a trick to the reuse the same API independently of the role a party plays.
	return context.RunView(nil, view.AsResponder(session), view.WithViewCall(
		func(context view.Context) (interface{}, error) {
			span := context.StartSpan("conversion_respond_view")
			defer span.End()
			// At some point, the recipient receives the token transaction that in the meantime has been assembled
			tx, err := ttx.ReceiveTransaction(context)
			assert.NoError(err, "failed to receive tokens")

			// The recipient can perform any check on the transaction as required by the business process
			// In particular, here, the recipient checks that the transaction contains at least one output, and
			// that there is at least one output that names the recipient.(The recipient is receiving something).
			outputs, err := tx.Outputs()
			assert.NoError(err, "failed getting outputs")
			assert.True(outputs.Count() > 0, "expected at least one output")
			assert.True(outputs.ByRecipient(id).Count() > 0, "expected at least one output assigned to [%s]", id)
			actualAmount := outputs.ByRecipient(id).Sum().Uint64()
			assert.True(actualAmount == i.Amount, "expected outputs to sum to [%d], got [%d]", i.Amount, actualAmount)

			// If everything is fine, the recipient accepts and sends back her signature.
			// Notice that, a signature from the recipient might or might not be required to make the transaction valid.
			// This depends on the driver implementation.
			span.AddEvent("accept_conversion")
			_, err = context.RunView(ttx.NewAcceptView(tx))
			assert.NoError(err, "failed to accept new tokens")

			// Before completing, the recipient waits for finality of the transaction
			span.AddEvent("ask_for_finality")
			_, err = context.RunView(ttx.NewFinalityView(tx, ttx.WithTimeout(1*time.Minute)))
			assert.NoError(err, "new tokens were not committed")

			return tx.ID(), nil
		},
	))
}

type ConversionInitiatorViewFactory struct{}

func (p *ConversionInitiatorViewFactory) NewView(in []byte) (view.View, error) {
	f := &ConversionInitiatorView{Conversion: &Conversion{}}
	err := json.Unmarshal(in, f.Conversion)
	assert.NoError(err, "failed unmarshalling input")

	return f, nil
}

type ConversionResponderView struct {
	Auditor string
}

func (p *ConversionResponderView) Call(context view.Context) (interface{}, error) {
	span := context.StartSpan("conversion_responder_view")
	defer span.End()
	// First the issuer receives the conversion request
	conversionRequest, err := ttx.ReceiveConversionRequest(context)
	assert.NoError(err, "failed to receive conversion request")

	// Now we have an inversion of roles. The issuer becomes an initiator.
	// This is a trick to reuse the code used in IssueCashView
	return context.RunView(nil, view.AsInitiator(), view.WithViewCall(func(context view.Context) (interface{}, error) {
		// Before assembling the transaction, the issuer can perform any activity that best fits the business process.
		// In this example, if the token type is USD, the issuer checks that no more than 230 units of USD
		// have been issued already including the current request.
		// No check is performed for other types.
		wallet := token.GetManagementService(context, token.WithTMSID(conversionRequest.TMSID)).WalletManager().IssuerWallet("")
		assert.NotNil(wallet, "issuer wallet not found")

		// At this point, the issuer is ready to prepare the token transaction.
		// The issuer creates a new token transaction and specifies the auditor that must be contacted to approve the operation.
		var tx *ttx.Transaction
		var auditorID string
		if len(p.Auditor) == 0 {
			assert.NoError(GetKVS(context).Get("auditor", &auditorID), "failed to retrieve auditor id")
		} else {
			auditorID = p.Auditor
		}
		auditor := view2.GetIdentityProvider(context).Identity(auditorID)
		if !conversionRequest.NotAnonymous {
			// The issuer creates an anonymous transaction (for Fabric, this means that the resulting transaction will be signed using idemix),
			tx, err = ttx.NewAnonymousTransaction(context, ttx.WithAuditor(auditor), ttx.WithTMSID(conversionRequest.TMSID))
		} else {
			// The issuer creates a nominal transaction using the default identity
			tx, err = ttx.NewTransaction(context, nil, ttx.WithAuditor(auditor), ttx.WithTMSID(conversionRequest.TMSID))
		}
		assert.NoError(err, "failed creating issue transaction")

		// The issuer adds a new issue operation to the transaction following the instruction received
		err = tx.Convert(
			wallet,
			conversionRequest.RecipientData.Identity,
			conversionRequest.UnspendableTokens,
		)
		assert.NoError(err, "failed adding new issued token")

		// The issuer is ready to collect all the required signatures.
		// In this case, the issuer's and the auditor's signatures.
		// Invoke the Token Chaincode to collect endorsements on the Token Request and prepare the relative transaction.
		// This is all done in one shot running the following view.
		// Before completing, all recipients receive the approved transaction.
		// Depending on the token driver implementation, the recipient's signature might or might not be needed to make
		// the token transaction valid.
		_, err = context.RunView(ttx.NewCollectEndorsementsView(tx))
		assert.NoError(err, "failed to sign issue transaction for "+tx.ID())

		// Last but not least, the issuer sends the transaction for ordering and waits for transaction finality.
		_, err = context.RunView(ttx.NewOrderingAndFinalityWithTimeoutView(tx, 1*time.Minute))
		assert.NoError(err, "failed to commit issue transaction")

		return tx.ID(), nil
	}))
}
