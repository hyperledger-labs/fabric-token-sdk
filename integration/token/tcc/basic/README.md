# Token SDK, The Basics

In this Section, we will see examples of how to perform basic token operations like `issue`, `transfer`, `swap`,
and so on.

Depending on the operation, one or more parties will be involved. 
An issuer and a recipient, a sender and a recipient, and so on.
One or more auditors might or might not be involved in the approval of the token transaction.
This depends on the underlying token driver used and the specific use-case.
In the following, we will consider the existence of an auditor in the system.

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

Thanks to the interaction between the issuer and the recipient, the recipient
becomes aware that some tokens have been issued to her. 
Once the transaction is final, the is what the vault of each party will contain:
- The issuer's vault will contain a reference to the issued tokens that can be queried via `HistoryTokens`.
- The recipient's vault will contain a reference to the same tokens. The recipient can query the vault, 
  or the wallet used to derive the recipient identity. We will see examples in the coming sections.

## Transfer

Transfer is a business interactive protocol among at least two parties: a `sender` and one or more `recipients`.
We assume that both sender and recipient are running a `Fabric Smart Client` node.

Here is an example of a `view` representing the sender's operations in the `transfer process`:  
This view is execute by the sender's FSC node.

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

    // The sender will select tokens owned by this wallet
    senderWallet := ttxcc.GetWallet(context, t.Wallet)
    assert.NotNil(senderWallet, "sender wallet [%s] not found", t.Wallet)

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
        senderWallet,
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

The `view` representing the recipient's operations can be exactly the same of that used for the issuance, or different
It depends on the specific business process.

Thanks to the interaction between the sender and the recipient, the recipient
becomes aware that some tokens have been transfer to her.
Once the transaction is final, the is what the vault of each party will contain:
- The token spent will disappear form the sender's vault.
- The recipient's vault will contain a reference to the freshly created tokens originated from the transfer.
  (Don't forget, we use the UTXO model here)
  The recipient can query the vault,
  or the wallet used to derive the recipient identity. We will see examples in the coming sections.

Let us now explore another transfer with a more elaborated token selection process that handles a situation
where not enough tokens of a certain type might be available. 

```go
type TransferWithSelectorView struct {
	*Transfer
}

func (t *TransferWithSelectorView) Call(context view.Context) (interface{}, error) {
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

	// The sender will select tokens owned by this wallet
	senderWallet := ttxcc.GetWallet(context, t.Wallet)
	assert.NotNil(senderWallet, "sender wallet [%s] not found", t.Wallet)

	// If no specific tokens are requested, then a custom token selection process start
	if len(t.TokenIDs) == 0 {
		// The sender uses the default token selector each transaction comes equipped with
		selector, err := tx.Selector()
		assert.NoError(err, "failed getting token selector")

		// The sender tries to select the requested amount of tokens of the passed type.
		// If a failure happens, the sender retries up to 5 times, waiting 10 seconds after each failure.
		// This is just an example, any other policy can be implemented.
		var ids []*token.Id
		var sum token.Quantity

		for i := 0; i < 5; i++ {
			// Select the request amount of tokens of the given type
			ids, sum, err = selector.Select(
				ttxcc.GetWallet(context, t.Wallet),
				token.NewQuantityFromUInt64(t.Amount).Decimal(),
				t.Type,
			)
			// If an error occurs and retry has been asked, then wait first a bit
			if err != nil && t.Retry {
				time.Sleep(10 * time.Second)
				continue
			}
			break
		}
		if err != nil {
			// If finally not enough tokens were available, the sender can check what was the cause of the error:
			cause := errors.Cause(err)
			switch cause {
			case nil:
				assert.NoError(err, "system failure")
			case token2.SelectorInsufficientFunds:
				assert.NoError(err, "pineapple")
			case token2.SelectorSufficientButLockedFunds:
				assert.NoError(err, "lemonade")
			case token2.SelectorSufficientButNotCertifiedFunds:
				assert.NoError(err, "mandarin")
			case token2.SelectorSufficientFundsButConcurrencyIssue:
				assert.NoError(err, "peach")
			}
		}

		// If the sender reaches this point, it means that tokens are available.
		// The sender can further inspect these tokens, if the business logic requires to do so.
		// Here is an example. The sender double checks that the tokens selected are the expected

		// First, the sender queries the vault to get the tokens
		tokens, err := tx.TokenService().Vault().NewQueryEngine().GetTokens(ids...)
		assert.NoError(err, "failed getting tokens from ids")

		// Then, the sender double check that what returned by the selector is correct
		recomputedSum := token.NewQuantityFromUInt64(0)
		for _, tok := range tokens {
			// Is the token of the right type?
			assert.Equal(t.Type, tok.Type, "expected token of type [%s], got [%s]", t.Type, tok.Type)
			// Add the quantity to the total
			q, err := token.ToQuantity(tok.Quantity, 64)
			assert.NoError(err, "failed converting quantity")
			recomputedSum = recomputedSum.Add(q)
		}
		// Is the recomputed sum correct?
		assert.True(sum.Cmp(recomputedSum) == 0, "sums do not match")
		// Is the amount selected equal or larger than what requested?
		assert.False(sum.Cmp(token.NewQuantityFromUInt64(t.Amount)) < 0, "if this point is reached, funds are sufficients")

		t.TokenIDs = ids
	}

	// The sender adds a new transfer operation to the transaction following the instruction received.
	// Notice the use of `token2.WithTokenIDs(t.TokenIDs...)` to pass the token ids selected above.
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

## Redeem

Depending on the driver used, a sender can redeem tokens directly or by requesting redeem to an authorized redeemer. 
In the following, we assume that the sender redeems directly.
Moreover, we assume that the sender is running a `Fabric Smart Client` node.
This view is execute by the sender's FSC node.

```go
// Redeem contains the input information for a redeem
type Redeem struct {
	// Wallet is the identifier of the wallet that owns the tokens to redeem
	Wallet string
	// TokenIDs contains a list of token ids to redeem. If empty, tokens are selected on the spot.
	TokenIDs []*token.Id
	// Type of tokens to redeem
	Type string
	// Amount to redeem
	Amount uint64
}

type RedeemView struct {
	*Redeem
}

func (t *RedeemView) Call(context view.Context) (interface{}, error) {
	// The sender directly prepare the token transaction.
	// The sender creates an anonymous transaction (this means that the result Fabric transaction will be signed using idemix),
	// and specify the auditor that must be contacted to approve the operation.
	tx, err := ttxcc.NewAnonymousTransaction(
		context,
		ttxcc.WithAuditor(fabric.GetIdentityProvider(context).Identity("auditor")),
	)
	assert.NoError(err, "failed creating transaction")

	// The sender will select tokens owned by this wallet
	senderWallet := ttxcc.GetWallet(context, t.Wallet)
	assert.NotNil(senderWallet, "sender wallet [%s] not found", t.Wallet)

	// The sender adds a new redeem operation to the transaction following the instruction received.
	// Notice the use of `token2.WithTokenIDs(t.TokenIDs...)`. If t.TokenIDs is not empty, the Redeem
	// function uses those tokens, otherwise the tokens will be selected on the spot.
	// Token selection happens internally by invoking the default token selector:
	// selector, err := tx.TokenService().SelectorManager().NewSelector(tx.ID())
	// assert.NoError(err, "failed getting selector")
	// selector.Select(wallet, amount, tokenType)
	// It is also possible to pass a custom token selector to the Redeem function by using the relative opt:
	// token2.WithTokenSelector(selector).
	err = tx.Redeem(
		senderWallet,
		t.Type,
		t.Amount,
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

## Swap

Let us now consider a more comple scenario. Alice and Bob are two business parties that want to swap some of their assets/tokens.
For example, Alice sends 100 USD to Bob in exchange for 95 EUR. The swap of these assets must happen automatically.
Let us now present how such an operation can be orchestrated using the `Fabric Smart Client` and the `Fabric Token SDK`.
We assume that bob Alice and Bob running a `Fabric Smart Client` node, and Alice is the initiator of the swap.

This view is execute by Alice's FSC node to initiate the swap.

```go

// Swap contains the input information for a swap
type Swap struct {
	// AliceWallet is the wallet Alice will use
	AliceWallet string
	// FromAliceType is the token type Alice will transfer
	FromAliceType string
	// FromAliceAmount is the amount Alice will transfer
	FromAliceAmount uint64
	// FromBobType is the token type Bob will transfer
	FromBobType string
	// FromBobAmount is the amount Bob will transfer
	FromBobAmount uint64
	// Bob is the identity of the Bob's FSC node
	Bob view.Identity
}

type SwapInitiatorView struct {
	*Swap
}

func (t *SwapInitiatorView) Call(context view.Context) (interface{}, error) {
	// As a first step operation, Alice contacts the recipient's FSC node
	// to exchange identities to use to assign ownership of the transferred tokens.
	me, other, err := ttxcc.ExchangeRecipientIdentities(context, t.AliceWallet, t.Bob)
	assert.NoError(err, "failed exchanging identities")

	// At this point, Alice is ready to prepare the token transaction.
	// Alice creates an anonymous transaction (this means that the result Fabric transaction will be signed using idemix),
	// and specify the auditor that must be contacted to approve the operation.
	tx, err := ttxcc.NewAnonymousTransaction(
		context,
		ttxcc.WithAuditor(fabric.GetIdentityProvider(context).Identity("auditor")),
	)
	assert.NoError(err, "failed creating transaction")

	// Alice will select tokens owned by this wallet
	senderWallet := ttxcc.GetWallet(context, t.AliceWallet)
	assert.NotNil(senderWallet, "sender wallet [%s] not found", t.AliceWallet)

	// Alice adds a new transfer operation to the transaction following the instruction received.
	err = tx.Transfer(
		senderWallet,
		t.FromAliceType,
		[]uint64{t.FromAliceAmount},
		[]view.Identity{other},
	)
	assert.NoError(err, "failed adding output")

	// At this point, Alice is ready to collect Bob's transfer.
	// She does that by using the CollectActionsView.
	// Alice specifies the actions that she is expecting to be added to the transaction.
	// For each action, Alice contacts the recipient sending the transaction and the expected action.
	// At the end of the view, tx contains the collected actions
	_, err = context.RunView(ttxcc.NewCollectActionsView(tx,
		&ttxcc.ActionTransfer{
			From:      other,
			Type:      t.FromBobType,
			Amount:    t.FromBobAmount,
			Recipient: me,
		},
	))
	assert.NoError(err, "failed collecting actions")

	// Alice doubles check that the content of the transaction is the one expected.
	assert.NoError(tx.Verify(), "failed verifying transaction")

	outputs, err := tx.Outputs()
	assert.NoError(err, "failed getting outputs")
	os := outputs.ByRecipient(other)
	assert.Equal(0, os.Sum().Cmp(token2.NewQuantityFromUInt64(t.FromAliceAmount)))
	assert.Equal(os.Count(), os.ByType(t.FromAliceType).Count())

	os = outputs.ByRecipient(me)
	assert.Equal(0, os.Sum().Cmp(token2.NewQuantityFromUInt64(t.FromBobAmount)))
	assert.Equal(os.Count(), os.ByType(t.FromBobType).Count())

	// Alice is ready to collect all the required signatures and form the Fabric Transaction.
	_, err = context.RunView(ttxcc.NewCollectEndorsementsView(tx))
	assert.NoError(err, "failed to sign transaction")

	// Send to the ordering service and wait for finality
	_, err = context.RunView(ttxcc.NewOrderingAndFinalityView(tx))
	assert.NoError(err, "failed asking ordering")

	return tx.ID(), nil
}

```

Here is the `view` representing Bob's side of the swap process,  
This view is execute by Bob's FSC node upon a message received from Alice.

```go

type SwapResponderView struct {}

func (t *SwapResponderView) Call(context view.Context) (interface{}, error) {
	// As a first step, Bob responds to the request to exchange token recipient identities.
	// Bob takes his token recipient identity from the default wallet (ttxcc.MyWallet(context)),
	// if not otherwise specified.
	_, _, err := ttxcc.RespondExchangeRecipientIdentities(context)
	assert.NoError(err, "failed getting identity")

	// To respond to a call from the CollectActionsView, the first thing to do is to receive
	// the transaction and the requested action.
	// This could happen multiple times, depending on the use-case.
	tx, action, err := ttxcc.ReceiveAction(context)
	assert.NoError(err, "failed receiving action")

	// Depending on the use case, Bob can further analyse the requested action, before proceeding. It depends on the use-case.
	// If everything is fine, Bob adds his transfer to Alice as requested.
	// Bob will select tokens from his default wallet matching the transaction
	bobWallet := ttxcc.MyWalletFromTx(context, tx)
	assert.NotNil(bobWallet, "Bob's default wallet not found")
	err = tx.Transfer(
		bobWallet,
		action.Type,
		[]uint64{action.Amount},
		[]view.Identity{action.Recipient},
	)
	assert.NoError(err, "failed appending transfer")

	// Once Bob finishes the preparation of his part, he can send Back the transaction
	// calling the CollectActionsResponderView
	_, err = context.RunView(ttxcc.NewCollectActionsResponderView(tx, action))
	assert.NoError(err, "failed responding to action collect")

	// If everything is fine, Bob endorses and sends back his signature.
	_, err = context.RunView(ttxcc.NewEndorseView(tx))
	assert.NoError(err, "failed endorsing transaction")

	// Before completing, Bob waits for finality of the transaction
	_, err = context.RunView(ttxcc.NewFinalityView(tx))
	assert.NoError(err)

	return tx.ID(), nil
}
```

## Queries

Here are two examples of view to list tokens.

The following view returns the list of unspent tokens:

```go

// ListUnspentTokens contains the input to query the list of unspent tokens
type ListUnspentTokens struct {
	// Wallet whose identities own the token
	Wallet string
	// TokenType is the token type to select
	TokenType string
}

type ListUnspentTokensView struct {
	*ListUnspentTokens
}

func (p *ListUnspentTokensView) Call(context view.Context) (interface{}, error) {
	// Tokens owner by identities in this wallet will be listed
	wallet := ttxcc.GetWallet(context, p.Wallet)
	assert.NotNil(wallet, "wallet [%s] not found", p.Wallet)

	// Return the list of unspent tokens by type
	return wallet.ListUnspentTokens(ttxcc.WithType(p.TokenType))
}
```

This other view returns the list of issued tokens:

```go
// ListIssuedTokens contains the input to query the list of issued tokens
type ListIssuedTokens struct {
	// Wallet whose identities own the token
	Wallet string
	// TokenType is the token type to select
	TokenType string
}

type ListIssuedTokensView struct {
	*ListIssuedTokens
}

func (p *ListIssuedTokensView) Call(context view.Context) (interface{}, error) {
	// Tokens issued by identities in this wallet will be listed
	wallet := ttxcc.GetIssuerWallet(context, p.Wallet)
	assert.NotNil(wallet, "wallet [%s] not found", p.Wallet)

	// Return the list of issued tokens by type
	return wallet.ListIssuedTokens(ttxcc.WithType(p.TokenType))
}
```

## Testing

Now that we have familiarized with the Token SDK. 
We can ask the question: `How can I test the above views?`

To be continued...