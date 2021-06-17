/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package views

import (
	"encoding/json"
	"fmt"

	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxcc"
)

type RegisterIssuer struct {
	TokenTypes []string
}

type RegisterIssuerView struct {
	*RegisterIssuer
}

func (r *RegisterIssuerView) Call(context view.Context) (interface{}, error) {
	for _, tokenType := range r.TokenTypes {
		_, err := context.RunView(ttxcc.NewRegisterIssuerIdentityView(tokenType))
		assert.NoError(err, "failed registering issuer identity for token type [%s]", tokenType)
	}

	return nil, nil
}

type RegisterIssuerViewFactory struct{}

func (p *RegisterIssuerViewFactory) NewView(in []byte) (view.View, error) {
	f := &RegisterIssuerView{RegisterIssuer: &RegisterIssuer{}}
	err := json.Unmarshal(in, f.RegisterIssuer)
	assert.NoError(err, "failed unmarshalling input")

	return f, nil
}

// IssueCash contains the input information to issue a token
type IssueCash struct {
	// IssuerWallet is the issuer's wallet to use
	IssuerWallet string
	// TokenType is the type of token to issue
	TokenType string
	// Quantity represent the number of units of a certain token type stored in the token
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
	recipient, err := ttxcc.RequestRecipientIdentity(context, p.Recipient)
	assert.NoError(err, "failed getting recipient identity")

	// Before assembling the transaction, the issuer can perform any activity that best fits the business process.
	// In this example, if the token type is USD, the issuer checks that no more than 230 units of USD
	// have been issued already including the current request.
	// No check is performed for other types.
	wallet := ttxcc.GetIssuerWallet(context, p.IssuerWallet)
	if p.TokenType == "USD" {
		// Retrieve the list of issued tokens using a specific wallet for a given token type.
		history, err := wallet.HistoryTokens(ttxcc.WithType(p.TokenType))
		assert.NoError(err, "failed getting history for token type [%s]", p.TokenType)
		fmt.Printf("History [%s,%s]<[230]?\n", history.Sum(64).ToBigInt().Text(10), p.TokenType)

		// Fail if the sum of the issued tokens and the current quest is larger than 230
		assert.True(history.Sum(64).Add(token2.NewQuantityFromUInt64(p.Quantity)).Cmp(token2.NewQuantityFromUInt64(230)) <= 0)
	}

	// At this point, the issuer is ready to prepare the token transaction.
	// The issuer creates an anonymous transaction (this means that the result Fabric transaction will be signed using idemix),
	// and specify the auditor that must be contacted to approve the operation
	tx, err := ttxcc.NewAnonymousTransaction(
		context,
		ttxcc.WithAuditor(
			fabric.GetIdentityProvider(context).Identity("auditor"), // Retrieve the auditor's FSC node identity
		),
	)
	assert.NoError(err, "failed creating issue transaction")

	// The issuer add a new issue operation to the transaction following the instruction received
	err = tx.Issue(
		wallet,
		recipient,
		p.TokenType,
		p.Quantity,
	)
	assert.NoError(err, "failed adding new issued token")

	// The issuer is ready to collect all the required signatures.
	// In this case, the issuer's and the auditor's signatures.
	// Invoke the Token Chaincode to collect endorsements on the Token Request and prepare the relative Fabric transaction.
	// This is all done in one shot running the following view.
	// Before completing, all recipients receive the approved Fabric transaction.
	_, err = context.RunView(ttxcc.NewCollectEndorsementsView(tx))
	assert.NoError(err, "failed to sign issue transaction")

	// Last but not least, the issuer sends the transaction for ordering and waits for transaction finality.
	_, err = context.RunView(ttxcc.NewOrderingView(tx))
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
