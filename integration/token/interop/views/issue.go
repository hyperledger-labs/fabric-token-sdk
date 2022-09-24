/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package views

import (
	"encoding/json"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
)

// IssueCash contains the input information to issue a token
type IssueCash struct {
	// TMSID identifies the TMS to use to perform the token operation
	TMSID token.TMSID
	// IssuerWallet is the issuer's wallet to use
	IssuerWallet string
	// TokenType is the type of token to issue
	TokenType string
	// Quantity represents the number of units of a certain token type to issue
	Quantity uint64
	// Recipient is the identity of the recipient's FSC node
	Recipient view.Identity
}

type IssueCashView struct {
	*IssueCash
}

func (p *IssueCashView) Call(context view.Context) (interface{}, error) {
	// As a first step operation, the issuer contacts the recipient's FSC node
	// to ask for the identity to use to assign ownership of the freshly created token.
	// Notice that, this step would not be required if the issuer knows already which
	// identity the recipient wants to use.
	recipient, err := ttx.RequestRecipientIdentity(context, p.Recipient, token.WithTMSID(p.TMSID))
	assert.NoError(err, "failed getting recipient identity")

	// Before assembling the transaction, the issuer can perform any activity that best fits the business process.
	wallet := ttx.GetIssuerWallet(context, p.IssuerWallet, token.WithTMSID(p.TMSID))
	assert.NotNil(wallet, "issuer wallet [%s] not found", p.IssuerWallet)

	// At this point, the issuer is ready to prepare the token transaction.
	// The issuer creates a transaction and specify the auditor that must be contacted to approve the operation.
	tx, err := ttx.NewAnonymousTransaction(
		context,
		ttx.WithAuditor(
			view2.GetIdentityProvider(context).Identity("auditor"), // Retrieve the auditor's FSC node identity
		),
		ttx.WithTMSID(p.TMSID),
	)
	assert.NoError(err, "failed creating issue transaction")

	// The issuer adds a new issue operation to the transaction following the instruction received
	err = tx.Issue(
		wallet,
		recipient,
		p.TokenType,
		p.Quantity,
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

	// Last but not least, the issuer sends the transaction for ordering and waits for transaction finality
	_, err = context.RunView(ttx.NewOrderingAndFinalityView(tx))
	assert.NoError(err, "failed to commit issue transaction")

	return tx.ID(), nil
}

type IssueCashViewFactory struct{}

func (p *IssueCashViewFactory) NewView(in []byte) (view.View, error) {
	f := &IssueCashView{IssueCash: &IssueCash{}}
	err := json.Unmarshal(in, f.IssueCash)
	assert.NoError(err, "failed unmarshalling input")

	return f, nil
}
