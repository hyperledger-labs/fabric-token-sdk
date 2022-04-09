/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package views

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger-labs/fabric-token-sdk/token"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"

	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
)

// IssueCash contains the input information to issue a token
type IssueCash struct {
	// IssuerWallet is the issuer's wallet to use
	IssuerWallet string
	// TokenType is the type of token to issue
	TokenType string
	// Quantity represent the number of units of a certain token type stored in the token
	Quantity uint64
	// Recipient is an identifier of the recipient identity
	Recipient string
}

type IssueCashView struct {
	*IssueCash
}

func (p *IssueCashView) Call(context view.Context) (interface{}, error) {
	// As a first step operation, the issuer contacts the recipient's FSC node
	// to ask for the identity to use to assign ownership of the freshly created token.
	// Notice that, this step would not be required if the issuer knew already which
	// identity the recipient wants to use.
	recipient, err := ttx.RequestRecipientIdentity(context, view2.GetIdentityProvider(context).Identity(p.Recipient))
	assert.NoError(err, "failed getting recipient identity")

	// Before assembling the transaction, the issuer can perform any activity that best fits the business process.
	// In this example, if the token type is USD, the issuer checks that no more than 230 units of USD
	// have been issued already including the current request.
	// No check is performed for other types.
	wallet := ttx.GetIssuerWallet(context, p.IssuerWallet)
	assert.NotNil(wallet, "issuer wallet [%s] not found", p.IssuerWallet)
	if p.TokenType == "USD" {
		// Retrieve the list of issued tokens using a specific wallet for a given token type.
		precision := token.GetManagementService(context).PublicParametersManager().Precision()

		history, err := wallet.ListIssuedTokens(ttx.WithType(p.TokenType))
		assert.NoError(err, "failed getting history for token type [%s]", p.TokenType)
		fmt.Printf("History [%s,%s]<[230]?\n", history.Sum(precision).ToBigInt().Text(10), p.TokenType)

		// Fail if the sum of the issued tokens and the current quest is larger than 230
		assert.True(history.Sum(precision).Add(token2.NewQuantityFromUInt64(p.Quantity)).Cmp(token2.NewQuantityFromUInt64(230)) <= 0)
	}

	// At this point, the issuer is ready to prepare the token transaction.
	// The issuer creates an anonymous transaction (this means that the resulting Fabric transaction will be signed using idemix, for example),
	// and specify the auditor that must be contacted to approve the operation
	tx, err := ttx.NewAnonymousTransaction(
		context,
		ttx.WithAuditor(
			view2.GetIdentityProvider(context).Identity("auditor"), // Retrieve the auditor's FSC node identity
		),
	)
	tx.SetApplicationMetadata("github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/issue", []byte("issue"))
	tx.SetApplicationMetadata("github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/meta", []byte("meta"))
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

	// Last but not least, the issuer sends the transaction for ordering and waits for transaction finality.
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
