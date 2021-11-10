package workload

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxcc"
)

type Issue struct {
	// IssuerWallet is the issuer's wallet to use
	IssuerWallet string
	// TokenType is the type of token to issue
	TokenType string
	// Quantity represent the number of units of a certain token type stored in the token
	Quantity uint64
	// Recipient is the identity of the recipient's FSC node
	Recipient view.Identity
}

type IssueView struct {
	*Issue
}

func (t *IssueView) Call(context view.Context) (interface{}, error) {
	logger.Infof("issue new tokens [%s]", context.ID())

	recipient, err := ttxcc.RequestRecipientIdentity(context, t.Recipient)
	assert.NoError(err, "failed getting recipient identity")

	wallet := ttxcc.GetIssuerWallet(context, t.IssuerWallet)
	assert.NotNil(wallet, "issuer wallet [%s] not found", t.IssuerWallet)

	tx, err := ttxcc.NewAnonymousTransaction(
		context,
		ttxcc.WithAuditor(
			fabric.GetDefaultIdentityProvider(context).Identity("auditor"), // Retrieve the auditor's FSC node identity
		),
	)
	assert.NoError(err, "failed creating issue transaction")

	// The issuer adds a new issue operation to the transaction following the instruction received
	err = tx.Issue(
		wallet,
		recipient,
		t.TokenType,
		t.Quantity,
	)
	assert.NoError(err, "failed adding new issued token")

	_, err = context.RunView(ttxcc.NewCollectEndorsementsView(tx))
	assert.NoError(err, "failed to sign issue transaction")

	return tx, nil
}
