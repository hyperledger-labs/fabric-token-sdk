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
	// Notice that, this step would not be required if the issuer knew already which
	// identity the recipient wants to use.
	recipient, err := ttxcc.RequestRecipientIdentity(context, p.Recipient)
	assert.NoError(err, "failed getting recipient identity")

	// Before assembling the transaction, the issuer can perform any activity that best fits the business process.
	// In this example, if the token type is USD, the issuer checks that no more than 230 units of USD
	// have been issued already including the current request.
	// No check is performed for other types.
	wallet := ttxcc.GetIssuerWallet(context, p.IssuerWallet)
	assert.NotNil(wallet, "issuer wallet [%s] not found", p.IssuerWallet)
	if p.TokenType == "USD" {
		// Retrieve the list of issued tokens using a specific wallet for a given token type.
		history, err := wallet.ListIssuedTokens(ttxcc.WithType(p.TokenType))
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
			fabric.GetDefaultIdentityProvider(context).Identity("auditor"), // Retrieve the auditor's FSC node identity
		),
	)
	tx.SetApplicationMetadata("github.com/hyperledger-labs/fabric-token-sdk/integration/token/tcc/basic/issue", []byte("issue"))
	tx.SetApplicationMetadata("github.com/hyperledger-labs/fabric-token-sdk/integration/token/tcc/basic/meta", []byte("meta"))
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
	// Invoke the Token Chaincode to collect endorsements on the Token Request and prepare the relative Fabric transaction.
	// This is all done in one shot running the following view.
	// Before completing, all recipients receive the approved Fabric transaction.
	// Depending on the token driver implementation, the recipient's signature might or might not be needed to make
	// the token transaction valid.
	_, err = context.RunView(ttxcc.NewCollectEndorsementsView(tx))
	assert.NoError(err, "failed to sign issue transaction")

	// Sanity checks:
	// - the transaction is in busy state in the vault
	fns := fabric.GetFabricNetworkService(context, tx.Network())
	ch, err := fns.Channel(tx.Channel())
	assert.NoError(err, "failed to retrieve channel [%s]", tx.Channel())
	vc, _, err := ch.Vault().Status(tx.ID())
	assert.NoError(err, "failed to retrieve vault status for transaction [%s]", tx.ID())
	assert.Equal(fabric.Busy, vc, "transaction [%s] should be in busy state", tx.ID())

	vc, _, err = ch.Committer().Status(tx.ID())
	assert.NoError(err, "failed to retrieve vault status for transaction [%s]", tx.ID())
	assert.Equal(fabric.Busy, vc, "transaction [%s] should be in busy state", tx.ID())

	// Last but not least, the issuer sends the transaction for ordering and waits for transaction finality.
	_, err = context.RunView(ttxcc.NewOrderingAndFinalityView(tx))
	assert.NoError(err, "failed to commit issue transaction")

	// Sanity checks:
	// - the transaction is in valid state in the vault
	vc, _, err = ch.Vault().Status(tx.ID())
	assert.NoError(err, "failed to retrieve vault status for transaction [%s]", tx.ID())
	assert.Equal(fabric.Valid, vc, "transaction [%s] should be in valid state", tx.ID())
	vc, _, err = ch.Committer().Status(tx.ID())
	assert.NoError(err, "failed to retrieve vault status for transaction [%s]", tx.ID())
	assert.Equal(fabric.Valid, vc, "transaction [%s] should be in busy state", tx.ID())

	return tx.ID(), nil
}

type IssueCashViewFactory struct{}

func (p *IssueCashViewFactory) NewView(in []byte) (view.View, error) {
	f := &IssueCashView{IssueCash: &IssueCash{}}
	err := json.Unmarshal(in, f.IssueCash)
	assert.NoError(err, "failed unmarshalling input")

	return f, nil
}
