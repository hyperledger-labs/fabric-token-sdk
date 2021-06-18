# Token SDK, The Basics

In this Section, we will see examples of how to perform basic token operations like `issue`, `transfer`, `swap`,
and so on.

Each operation will always involve two parties. An issuer and a recipient, a sender and a recipient, and so on.

Let us then describe each token operation with examples:

## Issuance

Issuance is a business interactive protocol among two parties: an `issuer` of a given token type
and a `recipient` that will become the owner of the freshly created token.
We assume that both issuer and recipient are running a `Fabric Smart Client` node. 

Here is an example of a `view` representing the issuer's operations in the `issuance process`:  
This view is execute by the issuer's FSC node.

```go

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

	// Last but not least, the issuer sends the transaction for ordering and waits for transaction finality.
	_, err = context.RunView(ttxcc.NewOrderingView(tx))
	assert.NoError(err, "failed to commit issue transaction")

	return tx.ID(), nil
}
```

Here is the `view` representing the recipient's operations, instead.  
This view is execute by the recipient's FSC node upon a message received from the issuer.

```go

type AcceptCashView struct{}

func (a *AcceptCashView) Call(context view.Context) (interface{}, error) {
	// The recipient of a token (issued or transfer) responds, as first operation,
	// to a request for a recipient.
	// The recipient can do that by using the following code.
	// The recipient identity will be taken from the default wallet (ttxcc.MyWallet(context)), if not otherwise specified.
	id, err := ttxcc.RespondRequestRecipientIdentity(context)
	assert.NoError(err, "failed to respond to identity request")

	// At some point, the recipient receives the token transaction that in the mean time has been assembled
	tx, err := ttxcc.ReceiveTransaction(context)
	assert.NoError(err, "failed to receive tokens")

	// The recipient can perform any check on the transaction as required by the business process
	// In particular, here, the recipient checks that the transaction contains at least one output, and
	// that there is at least one output that names the recipient. (The recipient is receiving something.
	outputs, err := tx.Outputs()
	assert.NoError(err, "failed getting outputs")
	assert.True(outputs.Count() > 0)
	assert.True(outputs.ByRecipient(id).Count() > 0)

	// The recipient here is checking that, for each type of token she is receiving,
	// she does not hold already more than 3000 units of that type.
	// Just a fancy query to show the capabilities of the services we are using.
	for _, output := range outputs.ByRecipient(id).Outputs() {
		unspentTokens, err := ttxcc.MyWallet(context).ListTokens(ttxcc.WithType(output.Type))
		assert.NoError(err, "failed retrieving the unspent tokens for type [%s]", output.Type)
		assert.True(
			unspentTokens.Sum(64).Cmp(token2.NewQuantityFromUInt64(3000)) <= 0,
			"cannot have more than 3000 unspent quantity for type [%s]", output.Type,
		)
	}

	// If everything is fine, the recipient accepts and sends back her signature.
	// Notice that, a signature from the recipient might or might not be required to make the transaction valid.
	// This depends on the driver implementation.
	_, err = context.RunView(ttxcc.NewAcceptView(tx))
	assert.NoError(err, "failed to accept new tokens")

	// Before completing, the recipient waits for finality of the transaction
	_, err = context.RunView(ttxcc.NewFinalityView(tx))
	assert.NoError(err, "new tokens were not committed")

	return nil, nil
}
```

## Transfer


```go
// Transfer contains the input information for a transfer
type Transfer struct {
	// Wallet is the identifier of the wallet that owns the tokens to transfer
	Wallet string
	// TokenIDs contains a list of token ids to transfer. If empty, tokens are selected on the spot.
	TokenIDs []*token.Id
	// Type of tokens to transfer
	Type string
	// Amount to transfer
	Amount uint64
	// Recipient is the identity of the recipient's FSC node
	Recipient view.Identity
	// Retry tells if a retry must happen in case of a failure
	Retry bool
}

type TransferView struct {
	*Transfer
}

func (t *TransferView) Call(context view.Context) (interface{}, error) {
	// As a first step operation, the sender contacts the recipient's FSC node
	// to ask for the identity to use to assign ownership of the freshly created token.
	// Notice that, this step would not be required if the sender knew already which
	// identity the recipient wants to use.
	recipient, err := ttxcc.RequestRecipientIdentity(context, t.Recipient)
	assert.NoError(err, "failed getting recipient")

	// At this point, the sender is ready to prepare the token transaction.
	// The sender creates an anonymous transaction (this means that the result Fabric transaction will be signed using idemix),
	// and specify the auditor that must be contacted to approve the operation.
	tx, err := ttxcc.NewAnonymousTransaction(
		context,
		ttxcc.WithAuditor(fabric.GetIdentityProvider(context).Identity("auditor")),
	)
	assert.NoError(err, "failed creating transaction")

	// The senders adds a new transfer operation to the transaction following the instruction received.
	// Notice the use of `token2.WithTokenIDs(t.TokenIDs...)`. If t.TokenIDs is not empty, the Transfer
	// function uses those tokens, otherwise the tokens will be selected on the spot.
	// Token selection happens internally by invoking the default token selector:
	// selector, err := tx.TokenService().SelectorManager().NewSelector(tx.ID())
	// assert.NoError(err, "failed getting selector")
	// selector.Select(wallet, amount, tokenType)
	// It is also possible to pass a custom token selector to the Transfer function by using the relative opt:
	// token2.WithTokenSelector(selector).
	err = tx.Transfer(
		ttxcc.GetWallet(context, t.Wallet),
		t.Type,
		[]uint64{t.Amount},
		[]view.Identity{recipient},
		token2.WithTokenIDs(t.TokenIDs...),
	)
	assert.NoError(err, "failed adding new tokens")

	// The sender is ready to collect all the required signatures.
	// In this case, the sender's and the auditor's signatures.
	// Invoke the Token Chaincode to collect endorsements on the Token Request and prepare the relative Fabric transaction.
	// This is all done in one shot running the following view.
	// Before completing, all recipients receive the approved Fabric transaction.
	// Depending on the token driver implementation, the recipient's signature might or might not be needed to make
	// the token transaction valid.
	_, err = context.RunView(ttxcc.NewCollectEndorsementsView(tx))
	assert.NoError(err, "failed to sign transaction")

	// Send to the ordering service and wait for finality
	_, err = context.RunView(ttxcc.NewOrderingAndFinalityView(tx))
	assert.NoError(err, "failed asking ordering")

	return tx.ID(), nil
}
```

The `view` representing the recipient's operations can be exactly the same of that used for the issuance.
It depends on the specific business process.

## 