/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package views

import (
	"encoding/json"
	"time"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
)

type Withdrawal struct {
	// TMSID the token management service identifier
	TMSID token.TMSID
	// Wallet of the recipient of the cash to be issued
	Wallet string
	// Amount represent the number of units of a certain token type stored in the token
	Amount uint64
	// TokenType is the type of token to issue
	TokenType string
	// Issuer identifies the issuer
	Issuer string
	// Recipient information
	RecipientData *token.RecipientData
}

type WithdrawalInitiatorView struct {
	*Withdrawal
}

func (i *WithdrawalInitiatorView) Call(context view.Context) (interface{}, error) {
	var id view.Identity
	var session view.Session
	var err error
	if i.RecipientData != nil {
		assert.NoError(
			token.GetManagementService(
				context, token.WithTMSID(i.TMSID),
			).WalletManager().OwnerWallet(i.Wallet).RegisterRecipient(i.RecipientData.Identity, i.RecipientData.AuditInfo, i.RecipientData.Metadata),
			"failed to register remote recipient",
		)
		id, session, err = ttx.RequestWithdrawalForRecipient(context, view.Identity(i.Issuer), i.Wallet, i.TokenType, i.Amount, i.RecipientData, token.WithTMSID(i.TMSID))
	} else {
		id, session, err = ttx.RequestWithdrawal(context, view.Identity(i.Issuer), i.Wallet, i.TokenType, i.Amount, token.WithTMSID(i.TMSID))
	}
	assert.NoError(err, "failed to send withdrawal request")

	return context.RunView(nil, view.AsResponder(session), view.WithViewCall(
		func(context view.Context) (interface{}, error) {
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
			_, err = context.RunView(ttx.NewAcceptView(tx))
			assert.NoError(err, "failed to accept new tokens")

			// Before completing, the recipient waits for finality of the transaction
			_, err = context.RunView(ttx.NewFinalityWithTimeoutView(tx, 1*time.Minute))
			assert.NoError(err, "new tokens were not committed")

			return tx.ID(), nil
		},
	))
}

type WithdrawalInitiatorViewFactory struct{}

func (p *WithdrawalInitiatorViewFactory) NewView(in []byte) (view.View, error) {
	f := &WithdrawalInitiatorView{Withdrawal: &Withdrawal{}}
	err := json.Unmarshal(in, f.Withdrawal)
	assert.NoError(err, "failed unmarshalling input")

	return f, nil
}

type WithdrawalResponderView struct {
}

func (p *WithdrawalResponderView) Call(context view.Context) (interface{}, error) {
	issueRequest, err := ttx.ReceiveWithdrawalRequest(context)
	assert.NoError(err, "failed to receive withdrawal request")

	return context.RunView(nil, view.AsInitiator(), view.WithViewCall(func(context view.Context) (interface{}, error) {
		// Before assembling the transaction, the issuer can perform any activity that best fits the business process.
		// In this example, if the token type is USD, the issuer checks that no more than 230 units of USD
		// have been issued already including the current request.
		// No check is performed for other types.
		wallet := token.GetManagementService(context, token.WithTMSID(issueRequest.TMSID)).WalletManager().IssuerWallet("")
		assert.NotNil(wallet, "issuer wallet not found")

		// At this point, the issuer is ready to prepare the token transaction.
		// The issuer creates a new token transaction and specifies the auditor that must be contacted to approve the operation.
		var tx *ttx.Transaction
		var auditor view.Identity
		var auditorID string
		assert.NoError(kvs.GetService(context).Get("auditor", &auditorID), "failed to retrieve auditor id")
		auditor = view2.GetIdentityProvider(context).Identity(auditorID)
		// The issuer creates an anonymous transaction (this means that the resulting Fabric transaction will be signed using idemix, for example),
		tx, err = ttx.NewAnonymousTransaction(context, ttx.WithAuditor(auditor), ttx.WithTMSID(issueRequest.TMSID))
		assert.NoError(err, "failed creating issue transaction")

		// The issuer adds a new issue operation to the transaction following the instruction received
		err = tx.Issue(
			wallet,
			issueRequest.Recipient,
			issueRequest.TokenType,
			issueRequest.Amount,
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
		assert.NoError(err, "failed to sign issue transaction")

		// Last but not least, the issuer sends the transaction for ordering and waits for transaction finality.
		_, err = context.RunView(ttx.NewOrderingAndFinalityWithTimeoutView(tx, 1*time.Minute))
		assert.NoError(err, "failed to commit issue transaction")

		return tx.ID(), nil
	}))
}
