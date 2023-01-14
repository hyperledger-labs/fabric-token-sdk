# Token SDK, Fungible Tokens, The Basics

In this Section, we will see examples of how to perform basic token operations like `issue`, `transfer`, `swap`,
and so on, on `fungible tokens`.

We will consider the following business parties:
- `Issuer`: The entity that creates/mints/issues the tokens.
- `Alice`, `Bob`, and `Charlie`: Each of these parties is a `fungible token` holder.
- `Auditor`: The entity that is auditing the token transactions.

Each party is running a Smart Fabric Client node with the Token SDK enabled.
The parties are connected in a peer-to-peer network that is established and maintained by the nodes.

**Remark**: The Smart Fabric Client SDK and the Token SDK can be embedded in an already existing 
Go-based application node by using the following code:
```go
  // import
  // 	fscnode "github.com/hyperledger-labs/fabric-smart-client/node"
  // "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/sdk"
  // sdk "github.com/hyperledger-labs/fabric-token-sdk/token/sdk"
    
  // Instantiate a new Fabric Smart Client Node
  n := fscnode.New() // Use NewFromConfPath(<conf-path>) to load configuration from a file
  // Install the required SDKs
  n.InstallSDK(fabric.NewSDK(n))
  n.InstallSDK(sdk.NewSDK(n))

  // Execute the stack
  n.Execute(func() error {
    // Your initialization code here
    return nil
  })
```
This is also the code that is executed by a standalone Fabric Smart Client node.

Now, let us then describe each token operation with examples:

## Issuance

Issuance is a business interactive protocol among two parties: an `issuer` of a given token type
and a `recipient` that will become the owner of the freshly created token.

Here is an example of a `view` representing the issuer's operations in the `issuance process`:  
This view is executed by the Issuer's FSC node.

```go
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
	recipient, err := ttxcc.RequestRecipientIdentity(context, view.Identity(p.Recipient))
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
	// The issuer creates an anonymous transaction (this means that the resulting transaction will be signed using idemix, for example),
	// and specify the auditor that must be contacted to approve the operation
	tx, err := ttxcc.NewAnonymousTransaction(
		context,
		ttxcc.WithAuditor(
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
	_, err = context.RunView(ttxcc.NewCollectEndorsementsView(tx))
	assert.NoError(err, "failed to sign issue transaction")

	// Last but not least, the issuer sends the transaction for ordering and waits for transaction finality.
	_, err = context.RunView(ttxcc.NewOrderingAndFinalityView(tx))
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
	// The recipient identity will be taken from the default wallet (ttx.MyWallet(context)), if not otherwise specified.
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
		unspentTokens, err := ttxcc.MyWallet(context).ListUnspentTokens(ttxcc.WithType(output.Type))
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
Once the transaction is final, this is what the vault of each party will contain:
- The issuer's vault will contain a reference to the issued tokens.
- The recipient's vault will contain a reference to the same tokens. The recipient can query the vault,
  or the wallet used to derive the recipient identity. We will see examples in the coming sections.

## Transfer

Transfer is a business interactive protocol among at least two parties: a `sender` and one or more `recipients`.

Here is an example of a `view` representing the sender's operations in the `transfer process`:  
This view is execute by the sender's FSC node.

```go
// Transfer contains the input information for a transfer
type Transfer struct {
	// Wallet is the identifier of the wallet that owns the tokens to transfer
	Wallet string
	// TokenIDs contains a list of token ids to transfer. If empty, tokens are selected on the spot.
	TokenIDs []*token.ID
	// TokenType of tokens to transfer
	TokenType string
	// Quantity to transfer
	Quantity uint64
	// Recipient is the identity of the recipient's FSC node
	Recipient string
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
	recipient, err := ttxcc.RequestRecipientIdentity(context, view.Identity(t.Recipient))
	assert.NoError(err, "failed getting recipient")

	// At this point, the sender is ready to prepare the token transaction.
	// The sender creates an anonymous transaction (this means that the resulting transaction will be signed using idemix, for example),
	// and specify the auditor that must be contacted to approve the operation.
	tx, err := ttxcc.NewAnonymousTransaction(
		context,
		ttxcc.WithAuditor(view2.GetIdentityProvider(context).Identity("auditor")),
	)
	assert.NoError(err, "failed creating transaction")

	// The sender will select tokens owned by this wallet
	senderWallet := ttxcc.GetWallet(context, t.Wallet)
	assert.NotNil(senderWallet, "sender wallet [%s] not found", t.Wallet)

	// The sender adds a new transfer operation to the transaction following the instruction received.
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
		t.TokenType,
		[]uint64{t.Quantity},
		[]view.Identity{recipient},
		token2.WithTokenIDs(t.TokenIDs...),
	)
	assert.NoError(err, "failed adding new tokens")

	// The sender is ready to collect all the required signatures.
	// In this case, the sender's and the auditor's signatures.
	// Invoke the Token Chaincode to collect endorsements on the Token Request and prepare the relative transaction.
	// This is all done in one shot running the following view.
	// Before completing, all recipients receive the approved transaction.
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

The `view` representing the recipient's operations can be exactly the same of that used for the issuance, or different.
It depends on the specific business process.

Thanks to the interaction between the sender and the recipient, the recipient
becomes aware that some tokens have been transfer to her.
Once the transaction is final, the is what the vault of each party will contain:
- The token spent will disappear form the sender's vault.
- The recipient's vault will contain a reference to the freshly created tokens originated from the transfer.
  (Don't forget, we use the UTXO model here)
  The recipient can query the vault,
  or the wallet used to derive the recipient identity. We will see examples in the coming sections.

## Redeem

Depending on the driver used, a sender can redeem tokens directly or by requesting redeem to an authorized redeemer.
In the following, we assume that the sender redeems directly.
This view is execute by the Sender's FSC node.

```go
// Redeem contains the input information for a redeem operation
type Redeem struct {
	// Wallet is the identifier of the wallet that owns the tokens to redeem
	Wallet string
	// TokenIDs contains a list of token ids to redeem. If empty, tokens are selected on the spot.
	TokenIDs []*token.ID
	// TokenType of tokens to redeem
	TokenType string
	// Quantity to redeem
	Quantity uint64
}

type RedeemView struct {
	*Redeem
}

func (t *RedeemView) Call(context view.Context) (interface{}, error) {
	// The sender directly prepare the token transaction.
	// The sender creates an anonymous transaction (this means that the resulting transaction will be signed using idemix, for example),
	// and specify the auditor that must be contacted to approve the operation.
	tx, err := ttxcc.NewAnonymousTransaction(
		context,
		ttxcc.WithAuditor(view2.GetIdentityProvider(context).Identity("auditor")),
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
		t.TokenType,
		t.Quantity,
		token2.WithTokenIDs(t.TokenIDs...),
	)
	assert.NoError(err, "failed adding new tokens")

	// The sender is ready to collect all the required signatures.
	// In this case, the sender's and the auditor's signatures.
	// Invoke the Token Chaincode to collect endorsements on the Token Request and prepare the relative transaction.
	// This is all done in one shot running the following view.
	// Before completing, all recipients receive the approved transaction.
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

Let us now consider a more complex scenario. Alice and Bob are two business parties that want to swap some of their assets/tokens.
For example, Alice sends 1 TOK to Bob in exchange for 1 KOT. The swap of these assets must happen automatically.

This view is execute by Alice's FSC node to initiate the swap.

```go
// Swap contains the input information for a swap
type Swap struct {
	// FromWallet is the wallet A will use
	FromWallet string
	// FromType is the token type A will transfer
	FromType string
	// FromQuantity is the amount A will transfer
	FromQuantity uint64
	// ToType is the token type To will transfer
	ToType string
	// ToQuantity is the amount To will transfer
	ToQuantity uint64
	// To is the identity of the To's FSC node
	To string
}

type SwapInitiatorView struct {
	*Swap
}

func (t *SwapInitiatorView) Call(context view.Context) (interface{}, error) {
	// As a first step operation, A contacts the recipient's FSC node
	// to exchange identities to use to assign ownership of the transferred tokens.
	me, other, err := ttxcc.ExchangeRecipientIdentities(context, t.FromWallet, view.Identity(t.To))
	assert.NoError(err, "failed exchanging identities")

	// At this point, A is ready to prepare the token transaction.
	// A creates an anonymous transaction (this means that the resulting transaction will be signed using idemix, for example),
	// and specify the auditor that must be contacted to approve the operation.
	tx, err := ttxcc.NewAnonymousTransaction(
		context,
		ttxcc.WithAuditor(view2.GetIdentityProvider(context).Identity("auditor")),
	)
	assert.NoError(err, "failed creating transaction")

	// A will select tokens owned by this wallet
	senderWallet := ttxcc.GetWallet(context, t.FromWallet)
	assert.NotNil(senderWallet, "sender wallet [%s] not found", t.FromWallet)

	// A adds a new transfer operation to the transaction following the instruction received.
	err = tx.Transfer(
		senderWallet,
		t.FromType,
		[]uint64{t.FromQuantity},
		[]view.Identity{other},
	)
	assert.NoError(err, "failed adding output")

	// At this point, A is ready to collect To's transfer.
	// She does that by using the CollectActionsView.
	// A specifies the actions that she is expecting to be added to the transaction.
	// For each action, A contacts the recipient sending the transaction and the expected action.
	// At the end of the view, tx contains the collected actions
	_, err = context.RunView(ttxcc.NewCollectActionsView(tx,
		&ttxcc.ActionTransfer{
			From:      other,
			Type:      t.ToType,
			Amount:    t.ToQuantity,
			Recipient: me,
		},
	))
	assert.NoError(err, "failed collecting actions")

	// A doubles check that the content of the transaction is the one expected.
	assert.NoError(tx.Verify(), "failed verifying transaction")

	outputs, err := tx.Outputs()
	assert.NoError(err, "failed getting outputs")
	os := outputs.ByRecipient(other)
	assert.Equal(0, os.Sum().Cmp(token2.NewQuantityFromUInt64(t.FromQuantity)))
	assert.Equal(os.Count(), os.ByType(t.FromType).Count())

	os = outputs.ByRecipient(me)
	assert.Equal(0, os.Sum().Cmp(token2.NewQuantityFromUInt64(t.ToQuantity)))
	assert.Equal(os.Count(), os.ByType(t.ToType).Count())

	// A is ready to collect all the required signatures and form the Transaction.
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
type SwapResponderView struct{}

func (t *SwapResponderView) Call(context view.Context) (interface{}, error) {
	// As a first step, To responds to the request to exchange token recipient identities.
	// To takes his token recipient identity from the default wallet (ttx.MyWallet(context)),
	// if not otherwise specified.
	_, _, err := ttxcc.RespondExchangeRecipientIdentities(context)
	assert.NoError(err, "failed getting identity")

	// To respond to a call from the CollectActionsView, the first thing to do is to receive
	// the transaction and the requested action.
	// This could happen multiple times, depending on the use-case.
	tx, action, err := ttxcc.ReceiveAction(context)
	assert.NoError(err, "failed receiving action")

	// Depending on the use case, To can further analyse the requested action, before proceeding. It depends on the use-case.
	// If everything is fine, To adds his transfer to A as requested.
	// To will select tokens from his default wallet matching the transaction
	bobWallet := ttxcc.MyWalletFromTx(context, tx)
	assert.NotNil(bobWallet, "To's default wallet not found")
	err = tx.Transfer(
		bobWallet,
		action.Type,
		[]uint64{action.Amount},
		[]view.Identity{action.Recipient},
	)
	assert.NoError(err, "failed appending transfer")

	// Once To finishes the preparation of his part, he can send Back the transaction
	// calling the CollectActionsResponderView
	_, err = context.RunView(ttxcc.NewCollectActionsResponderView(tx, action))
	assert.NoError(err, "failed responding to action collect")

	// If everything is fine, To endorses and sends back his signature.
	_, err = context.RunView(ttxcc.NewEndorseView(tx))
	assert.NoError(err, "failed endorsing transaction")

	// Before completing, the recipient waits for finality of the transaction
	_, err = context.RunView(ttxcc.NewFinalityView(tx))
	assert.NoError(err, "new tokens were not committed")

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

To run the `Fungible Tokens` sample, one needs first to deploy the `Fabric Smart Client` and the `Fabric` networks.
Once these networks are deployed, one can invoke views on the smart client nodes to test the `Fungible Tokens` sample.

So, first step is to describe the topology of the networks we need.

### Describe the topology of the networks

To test the above views, we have to first clarify the topology of the networks we need.
Namely, Fabric and FSC networks.

For Fabric, we will use a simple topology with:
1. Two organization: Org1 and Org2;
2. Single channel;
2. Org1 runs/endorse the Token Chaincode.

For the FSC network, we have a topology with a node for each business party.
1. Issuer and Auditor have an Org1 Fabric Identity;
2. Alice, Bob, and Charlie have an Org2 Fabric Identity.

We can describe the network topology programmatically as follows:

```go
func Topology(tokenSDKDriver string) []api.Topology {
	// Fabric
	fabricTopology := fabric.NewDefaultTopology()
	fabricTopology.EnableIdemix()
	fabricTopology.AddOrganizationsByName("Org1", "Org2")
	fabricTopology.SetNamespaceApproverOrgs("Org1")

	// FSC
	fscTopology := fsc.NewTopology()
	// fscTopology.SetLogging("grpc=error:debug", "")

	// issuer
	issuer := fscTopology.AddNodeByName("issuer").AddOptions(
		fabric.WithOrganization("Org1"),
		fabric.WithAnonymousIdentity(),
		token.WithDefaultIssuerIdentity(),
		token.WithIssuerIdentity("issuer.id1"),
	)
	issuer.RegisterViewFactory("issue", &views.IssueCashViewFactory{})
	issuer.RegisterViewFactory("issued", &views.ListIssuedTokensViewFactory{})

	// auditor
	auditor := fscTopology.AddNodeByName("auditor").AddOptions(
		fabric.WithOrganization("Org1"),
		fabric.WithAnonymousIdentity(),
		token.WithAuditorIdentity(),
	)
	auditor.RegisterViewFactory("registerAuditor", &views.RegisterAuditorViewFactory{})

	// alice
	alice := fscTopology.AddNodeByName("alice").AddOptions(
		fabric.WithOrganization("Org2"),
		fabric.WithAnonymousIdentity(),
		token.WithDefaultOwnerIdentity(tokenSDKDriver),
		token.WithOwnerIdentity(tokenSDKDriver, "alice.id1"),
	)
	alice.RegisterResponder(&views.AcceptCashView{}, &views.IssueCashView{})
	alice.RegisterResponder(&views.AcceptCashView{}, &views.TransferView{})
	alice.RegisterViewFactory("transfer", &views.TransferViewFactory{})
	alice.RegisterViewFactory("redeem", &views.RedeemViewFactory{})
	alice.RegisterViewFactory("swap", &views.SwapInitiatorViewFactory{})
	alice.RegisterViewFactory("unspent", &views.ListUnspentTokensViewFactory{})

	// bob
	bob := fscTopology.AddNodeByName("bob").AddOptions(
		fabric.WithOrganization("Org2"),
		fabric.WithAnonymousIdentity(),
		token.WithDefaultOwnerIdentity(tokenSDKDriver),
		token.WithOwnerIdentity(tokenSDKDriver, "bob.id1"),
	)
	bob.RegisterResponder(&views.AcceptCashView{}, &views.IssueCashView{})
	bob.RegisterResponder(&views.AcceptCashView{}, &views.TransferView{})
	bob.RegisterResponder(&views.SwapResponderView{}, &views.SwapInitiatorView{})
	bob.RegisterViewFactory("transfer", &views.TransferViewFactory{})
	bob.RegisterViewFactory("redeem", &views.RedeemViewFactory{})
	bob.RegisterViewFactory("swap", &views.SwapInitiatorViewFactory{})
	bob.RegisterViewFactory("unspent", &views.ListUnspentTokensViewFactory{})

	// charlie
	charlie := fscTopology.AddNodeByName("charlie").AddOptions(
		fabric.WithOrganization("Org2"),
		fabric.WithAnonymousIdentity(),
		token.WithDefaultOwnerIdentity(tokenSDKDriver),
		token.WithOwnerIdentity(tokenSDKDriver, "charlie.id1"),
	)
	charlie.RegisterResponder(&views.AcceptCashView{}, &views.IssueCashView{})
	charlie.RegisterResponder(&views.AcceptCashView{}, &views.TransferView{})
	charlie.RegisterResponder(&views.SwapResponderView{}, &views.SwapInitiatorView{})
	charlie.RegisterViewFactory("transfer", &views.TransferViewFactory{})
	charlie.RegisterViewFactory("redeem", &views.RedeemViewFactory{})
	charlie.RegisterViewFactory("swap", &views.SwapInitiatorViewFactory{})
	charlie.RegisterViewFactory("unspent", &views.ListUnspentTokensViewFactory{})

	tokenTopology := token.NewTopology()
	tokenTopology.SetSDK(fscTopology, &sdk.SDK{})
	tms := tokenTopology.AddTMS(fabricTopology, tokenSDKDriver)
	tms.SetNamespace([]string{"Org1"}, "100", "2")
    tms.AddAuditor(auditor)

	return []api.Topology{fabricTopology, tokenTopology, fscTopology}
}
```

The above topology takes in input the token driver name.

### Boostrap the networks

Bootstrap of the networks requires both Fabric Docker images and Fabric binaries. To ensure you have the required images you can use the following Makefile target in the project root directory:

```shell
make fabric-docker-images
```

To ensure you have the required fabric binary files and set the `FAB_BINS` environment variable to the correct place you can do the following in the project root directory

```shell
make download-fabric
export FAB_BINS=$PWD/../fabric/bin
```

To help us bootstrap the networks and then invoke the business views, the `fungible` command line tool is provided.
To build it, we need to run the following command from the folder `$GOPATH/src/github.com/hyperledger-labs/fabric-token-sdk/samples/fabric/fungible`.

```shell
go build -o fungible
```

If the compilation is successful, we can run the `fungible` command line tool as follows:

``` 
./fungible network start --path ./testdata
```

The above command will start the Fabric network and the FSC network,
and store all configuration files under the `./testdata` directory.
The CLI will also create the folder `./cmd` that contains a go main file for each FSC node. 
These go main files  are synthesized on the fly, and
the CLI compiles and then runs them.

If everything is successful, you will see something like the following:

```shell
2022-02-09 14:17:06.705 UTC [nwo.network] Start -> INFO 032  _____   _   _   ____
2022-02-09 14:17:06.705 UTC [nwo.network] Start -> INFO 033 | ____| | \ | | |  _ \
2022-02-09 14:17:06.705 UTC [nwo.network] Start -> INFO 034 |  _|   |  \| | | | | |
2022-02-09 14:17:06.705 UTC [nwo.network] Start -> INFO 035 | |___  | |\  | | |_| |
2022-02-09 14:17:06.705 UTC [nwo.network] Start -> INFO 036 |_____| |_| \_| |____/
2022-02-09 14:17:06.705 UTC [fsc.integration] Serve -> INFO 037 All GOOD, networks up and running...
2022-02-09 14:17:06.705 UTC [fsc.integration] Serve -> INFO 038 If you want to shut down the networks, press CTRL+C
2022-02-09 14:17:06.705 UTC [fsc.integration] Serve -> INFO 039 Open another terminal to interact with the networks
```

To shut down the networks, just press CTRL-C.

If you want to restart the networks after the shutdown, you can just re-run the above command.
If you don't delete the `./testdata` directory, the network will be started from the previous state.

Before restarting the networks, one can modify the business views to add new functionalities, to fix bugs, and so on.
Upon restarting the networks, the new business views will be available.
Later on, we will see an example of this.

To clean up all artifacts, we can run the following command:

```shell
./fungible network clean --path ./testdata
```

The `./testdata` and `./cmd` folders will be deleted.

### Anatomy of the configuration file

Each FSC node has a configuration file that contains the information about the FSC node itself,
the networks (Fabric, Orion, ...) that the FSC node is part of, and the token-related configuration.

Here is how the configuration file looks like for Alice's node:

```yaml
# Logging section
logging:
  # Spec
  spec: info
  # Format
  format: '%{color}%{time:2006-01-02 15:04:05.000 MST} [%{module}] %{shortfunc} -> %{level:.4s} %{id:03x}%{color:reset} %{message}'
# The fsc section is dedicated to the FSC node itself
fsc:
  # The FSC id provides a name for this node instance and is used when
  # naming docker resources.
  id: fsc.alice
  # The networkId allows for logical separation of networks and is used when
  # naming docker resources.
  networkId: u6l7g33mhjdn3ko7zrnzl5ftae
  # This represents the endpoint to other FSC nodes in the same organization.
  address: 127.0.0.1:20006
  # Whether the FSC node should programmatically determine its address
  # This case is useful for docker containers.
  # When set to true, will override FSC address.
  addressAutoDetect: true
  # GRPC Server listener address   
  listenAddress: 127.0.0.1:20006
  # Identity of this node, used to connect to other nodes
  identity:
    # X.509 certificate used as identity of this node
    cert:
      file: /home/vagrant/testdata/fsc/crypto/peerOrganizations/fsc.example.com/peers/alice.fsc.example.com/msp/signcerts/alice.fsc.example.com-cert.pem
    # Private key matching the X.509 certificate
    key:
      file: /home/vagrant/testdata/fsc/crypto/peerOrganizations/fsc.example.com/peers/alice.fsc.example.com/msp/keystore/priv_sk
  # Admin X.509 certificates
  admin:
    certs:
      - /home/vagrant/testdata/fsc/crypto/peerOrganizations/fsc.example.com/users/Admin@fsc.example.com/msp/signcerts/Admin@fsc.example.com-cert.pem
  # TLS Settings
  # (We use here the same set of properties as Hyperledger Fabric)
  tls:
    # Require server-side TLS
    enabled:  true
    # Require client certificates / mutual TLS for inbound connections.
    # Note that clients that are not configured to use a certificate will
    # fail to connect to the node.
    clientAuthRequired: false
    # X.509 certificate used for TLS server
    cert:
      file: /home/vagrant/testdata/fsc/crypto/peerOrganizations/fsc.example.com/peers/alice.fsc.example.com/tls/server.crt
    # Private key used for TLS server
    key:
      file: /home/vagrant/testdata/fsc/crypto/peerOrganizations/fsc.example.com/peers/alice.fsc.example.com/tls/server.key
    # X.509 certificate used for TLS when making client connections.
    # If not set, fsc.tls.cert.file will be used instead
    clientCert:
      file: /home/vagrant/testdata/fsc/crypto/peerOrganizations/fsc.example.com/peers/alice.fsc.example.com/tls/server.crt
    # Private key used for TLS when making client connections.
    # If not set, fsc.tls.key.file will be used instead
    clientKey:
      file: /home/vagrant/testdata/fsc/crypto/peerOrganizations/fsc.example.com/peers/alice.fsc.example.com/tls/server.key
    # rootcert.file represents the trusted root certificate chain used for verifying certificates
    # of other nodes during outbound connections.
    rootcert:
      file: /home/vagrant/testdata/fsc/crypto/peerOrganizations/fsc.example.com/peers/alice.fsc.example.com/tls/ca.crt
    # If mutual TLS is enabled, clientRootCAs.files contains a list of additional root certificates
    # used for verifying certificates of client connections.
    clientRootCAs:
      files:
        - /home/vagrant/testdata/fsc/crypto/peerOrganizations/fsc.example.com/peers/alice.fsc.example.com/tls/ca.crt
    rootCertFile: /home/vagrant/testdata/fsc/crypto/ca-certs.pem
  # Keepalive settings for node server and clients
  keepalive:
    # MinInterval is the minimum permitted time between client pings.
    # If clients send pings more frequently, the peer server will
    # disconnect them
    minInterval: 60s
    # Interval is the duration after which if the server does not see
    # any activity from the client it pings the client to see if it's alive
    interval: 300s
    # Timeout is the duration the server waits for a response
    # from the client after sending a ping before closing the connection
    timeout: 600s
  # P2P configuration
  p2p:
    # Listening address
    listenAddress: /ip4/127.0.0.1/tcp/20007
    # If empty, this is a P2P boostrap node. Otherwise, it contains the name of the FSC node that is a bootstrap node.
    # The name of the FSC node that is a bootstrap node must be set under fsc.endpoint.resolvers
    bootstrapNode: issuer
  # The Key-Value Store is used to store various information related to the FSC node
  kvs:
    persistence:
      # Persistence type can be \'badger\' (on disk) or \'memory\'
      type: badger
      opts:
        path: /home/vagrant/testdata/fsc/nodes/alice/kvs
  # HTML Server configuration for REST calls
  web:
    # Enable the REST server
    enabled: true
    # HTTPS server listener address
    address: 0.0.0.0:20008
    tls:
      # Require server-side TLS
      enabled:  true
      cert:
        file: /home/vagrant/testdata/fsc/crypto/peerOrganizations/fsc.example.com/peers/alice.fsc.example.com/tls/server.crt
      key:
        file: /home/vagrant/testdata/fsc/crypto/peerOrganizations/fsc.example.com/peers/alice.fsc.example.com/tls/server.key
      clientRootCAs:
        files:
          - /home/vagrant/testdata/fsc/crypto/peerOrganizations/fsc.example.com/peers/alice.fsc.example.com/tls/ca.crt
    rootCertFile: /home/vagrant/testdata/fsc/crypto/ca-certs.pem
  tracing:
    provider: none
    udp:
      address: 127.0.0.1:8125
  metrics:
    # metrics provider is one of statsd, prometheus, or disabled
    provider: prometheus
    # statsd configuration
    statsd:
      # network type: tcp or udp
      network: udp
      # statsd server address
      address: 127.0.0.1:8125
      # the interval at which locally cached counters and gauges are pushed
      # to statsd; timings are pushed immediately
      writeInterval: 10s
      # prefix is prepended to all emitted statsd metrics
      prefix:

  # The endpoint section tells how to reach other FSC node in the network.
  # For each node, the name, the domain, the identity of the node, and its addresses must be specified.
  endpoint:
    resolvers:
      - name: issuer
        domain: fsc.example.com
        identity:
          id: issuer
          path: /home/vagrant/testdata/fsc/crypto/peerOrganizations/fsc.example.com/peers/issuer.fsc.example.com/msp/signcerts/issuer.fsc.example.com-cert.pem
        addresses:
          P2P: 127.0.0.1:20001
        aliases:
          - issuer.id1
      - name: auditor
        domain: fsc.example.com
        identity:
          id: auditor
          path: /home/vagrant/testdata/fsc/crypto/peerOrganizations/fsc.example.com/peers/auditor.fsc.example.com/msp/signcerts/auditor.fsc.example.com-cert.pem
        addresses:
        aliases:
      - name: bob
        domain: fsc.example.com
        identity:
          id: bob
          path: /home/vagrant/testdata/fsc/crypto/peerOrganizations/fsc.example.com/peers/bob.fsc.example.com/msp/signcerts/bob.fsc.example.com-cert.pem
        addresses:
        aliases:
          - bob.id1
      - name: charlie
        domain: fsc.example.com
        identity:
          id: charlie
          path: /home/vagrant/testdata/fsc/crypto/peerOrganizations/fsc.example.com/peers/charlie.fsc.example.com/msp/signcerts/charlie.fsc.example.com-cert.pem
        addresses:
        aliases:
          - charlie.id1
          - 
# The fabric section defines the configuration of the fabric networks.  
fabric: 
  enabled: true
  # The name of the network 
  default:
    # Is this the default network?
    default: true
    # BCCSP configuration for this fabric network. Similar to the equivalent section in Fabric peer configuration.
    BCCSP:
      Default: SW
      SW:
        Hash: SHA2
        Security: 256
        FileKeyStore:
          KeyStore:
    # The MSP config path of the default identity to connect to this network.
    mspConfigPath: /home/vagrant/testdata/fabric.default/crypto/peerOrganizations/org2.example.com/peers/alice.org2.example.com/msp
    # Local MSP ID of the default identity
    localMspId: Org2MSP
    # Cache size to use when handling idemix pseudonyms. If the value is larger than 0, the cache is enabled and
    # pseudonyms are generated in batches of the given size to be ready to be used.
    mspCacheSize: 500
    # Additional MSP identities that can be used to connect to this network.
    msps:
      - id: idemix # The id of the identity. 
        mspType: idemix # The type of the MSP.
        mspID: IdemixOrgMSP # The MSP ID.
        # The path to the MSP folder containing the cryptographic materials.
        path: /home/vagrant/testdata/fabric.default/crypto/peerOrganizations/org2.example.com/peers/alice.org2.example.com/extraids/idemix
    # TLS Settings
    tls:
      enabled:  true
      clientAuthRequired: false
      cert:
        file: /home/vagrant/testdata/fabric.default/crypto/peerOrganizations/org2.example.com/peers/alice.org2.example.com/tls/server.crt
      key:
        file: /home/vagrant/testdata/fabric.default/crypto/peerOrganizations/org2.example.com/peers/alice.org2.example.com/tls/server.key
      clientCert:
        file: /home/vagrant/testdata/fabric.default/crypto/peerOrganizations/org2.example.com/peers/alice.org2.example.com/tls/server.crt
      clientKey:
        file: /home/vagrant/testdata/fabric.default/crypto/peerOrganizations/org2.example.com/peers/alice.org2.example.com/tls/server.key
      rootcert:
        file: /home/vagrant/testdata/fabric.default/crypto/peerOrganizations/org2.example.com/peers/alice.org2.example.com/tls/ca.crt
      clientRootCAs:
        files:
          - /home/vagrant/testdata/fabric.default/crypto/peerOrganizations/org2.example.com/peers/alice.org2.example.com/tls/ca.crt
      rootCertFile: /home/vagrant/testdata/fabric.default/crypto/ca-certs.pem
    # List of orderer nodes this node can connect to. There must be at least one orderer node.
    orderers:
      - address: 127.0.0.1:20015
        connectionTimeout: 10s
        tlsEnabled: true
        tlsRootCertFile: /home/vagrant/testdata/fabric.default/crypto/ca-certs.pem
        serverNameOverride:
    # List of trusted peers this node can connect to. There must be at least one trusted peer.
    peers:
      - address: 127.0.0.1:20026
        connectionTimeout: 10s
        tlsEnabled: true
        tlsRootCertFile: /home/vagrant/testdata/fabric.default/crypto/ca-certs.pem
        serverNameOverride:
    # List of channels this node is aware of
    channels:
      - name: testchannel
        default: true
    # Configuration of the vault used to store the RW sets assembled by this node
    vault:
      persistence:
        type: file
        opts:
          path: /home/vagrant/testdata/fsc/nodes/alice/fabric.default/vault
    # The endpoint section tells how to reach other Fabric nodes in the network.
    # For each node, the name, the domain, the identity of the node, and its addresses must be specified.
    endpoint:
      resolvers:
        - name: Org1_peer_0
          domain: org1.example.com
          identity:
            id: Org1_peer_0
            mspType: bccsp
            mspID: Org1MSP
            path: /home/vagrant/testdata/fabric.default/crypto/peerOrganizations/org1.example.com/peers/Org1_peer_0.org1.example.com/msp/signcerts/Org1_peer_0.org1.example.com-cert.pem
          addresses:
            Listen: 127.0.0.1:20019
          aliases:
        - name: Org2_peer_0
          domain: org2.example.com
          identity:
            id: Org2_peer_0
            mspType: bccsp
            mspID: Org2MSP
            path: /home/vagrant/testdata/fabric.default/crypto/peerOrganizations/org2.example.com/peers/Org2_peer_0.org2.example.com/msp/signcerts/Org2_peer_0.org2.example.com-cert.pem
          addresses:
            Listen: 127.0.0.1:20026
          aliases:
        - name: issuer
          domain: org1.example.com
          identity:
            id: issuer
            mspType: bccsp
            mspID: Org1MSP
            path: /home/vagrant/testdata/fabric.default/crypto/peerOrganizations/org1.example.com/peers/issuer.org1.example.com/msp/signcerts/issuer.org1.example.com-cert.pem
          addresses:
          aliases:
            - issuer
            - issuer.id1
        - name: auditor
          domain: org1.example.com
          identity:
            id: auditor
            mspType: bccsp
            mspID: Org1MSP
            path: /home/vagrant/testdata/fabric.default/crypto/peerOrganizations/org1.example.com/peers/auditor.org1.example.com/msp/signcerts/auditor.org1.example.com-cert.pem
          addresses:
          aliases:
            - auditor
        - name: alice
          domain: org2.example.com
          identity:
            id: alice
            mspType: bccsp
            mspID: Org2MSP
            path: /home/vagrant/testdata/fabric.default/crypto/peerOrganizations/org2.example.com/peers/alice.org2.example.com/msp/signcerts/alice.org2.example.com-cert.pem
          addresses:
          aliases:
            - alice
        - name: bob
          domain: org2.example.com
          identity:
            id: bob
            mspType: bccsp
            mspID: Org2MSP
            path: /home/vagrant/testdata/fabric.default/crypto/peerOrganizations/org2.example.com/peers/bob.org2.example.com/msp/signcerts/bob.org2.example.com-cert.pem
          addresses:
          aliases:
            - bob
        - name: charlie
          domain: org2.example.com
          identity:
            id: charlie
            mspType: bccsp
            mspID: Org2MSP
            path: /home/vagrant/testdata/fabric.default/crypto/peerOrganizations/org2.example.com/peers/charlie.org2.example.com/msp/signcerts/charlie.org2.example.com-cert.pem
          addresses:
          aliases:
            - charlie

token:
  enabled: true
  # The Token Transaction DB is a database of audited records. It is used to track the
  # history of audited events. In particular, it is used to track payments, holdings,
  # and transactions of any business party identified by a unique enrollment ID.
  ttxdb:
    persistence:
      # The type of persistence mechanism used to store the Token Transaction DB. (memory, badger)
      type: badger
      opts:
        # The path to the Token Transaction DB.
        path: /home/vagrant/testdata/fsc/nodes/alice/kvs
  # TMS stands for Token Management Service. A TMS is uniquely identified by a network, channel, and 
  # namespace identifiers. The network identifier should refer to a configured network (Fabric, Orion, and so on).
  # The meaning of channel and namespace are network dependent. For Fabric, the meaning is clear.
  # For Orion, channel is empty and namespace is the DB name to use.
  tms:
    - channel: testchannel # Channel identifier within the specified network
      namespace: zkat # Namespace identifier within the specified channel
      # Network identifier this TMS refers to. It must match the identifier of a Fabric or Orion network
      network: default
      # Wallets associated with this TMS
      wallets:
        # Owners wallets are used to own tokens
        owners:
          - default: true
            # ID of the wallet
            id: alice 
            # Path to folder containing the cryptographic material 
            path: /home/vagrant/testdata/token/crypto/default-testchannel-zkat/idemix/alice
            # Type of the wallet in the form of <type>:<MSPID>:<idemix elliptic curve>
            type: idemix:IdemixOrgMSP:BN254
          - default: false
            id: alice.id1
            path: /home/vagrant/testdata/token/crypto/default-testchannel-zkat/idemix/alice.id1
            type: idemix:IdemixOrgMSP:BN254
```

### Invoke the business views

If you reached this point, you can now invoke the business views on the FSC nodes.

To issue a fungible token, we can run the following command:

```shell
./fungible view -c ./testdata/fsc/nodes/issuer/client-config.yaml -f issue -i "{\"TokenType\":\"TOK\", \"Quantity\":10, \"Recipient\":\"alice\"}"
```

The above command invoke the `issue` view on the issuer's FSC node. The `-c` option specifies the client configuration file.
The `-f` option specifies the view name. The `-i` option specifies the input data.
In the specific case, we are asking the issuer to issue a fungible token whose type is `TOK` and quantity is 10, and the recipient is `alice`.
If everything is successful, you will see something like the following:

```shell
"594fbed71f03879b95463d9f68bc6af2f221207840c4a943faa054f729e0752e"
```
The above is the transaction id of the transaction that issued the fungible token.

Indeed, once the token is issued, the recipient can query its wallet to see the token.

```shell
./fungible view -c ./testdata/fsc/nodes/alice/client-config.yaml -f unspent -i "{\"TokenType\":\"TOK\"}"
```

The above command will query Alice's wallet to get a list of unspent tokens whose type `TOK`.
You can expect to see an output like this (beautified): 
```shell
{
  "tokens": [
    {
      "Id": {
        "tx_id": "594fbed71f03879b95463d9f68bc6af2f221207840c4a943faa054f729e0752e"
      },
      "Owner": {
        "raw": "MIIEmxMCc2kEggSTCgxJZGVtaXhPcmdNU1ASggkKIAHCT4uQAPEP1883dXxsJYXshm+r/8Sl+KWKQBPtsDMtEiAN1qnd8QKF2YSv6Bftzt1v5XqH30yzJqlzp777FoBRsxpZCgxJZGVtaXhPcmdNU1ASJ2RlZmF1bHQtdGVzdGNoYW5uZWwtZGVmYXVsdC5leGFtcGxlLmNvbRogAvYocoRRzxnU/3a5uT0Lo/0S+q6nBi5/kLf96ay1gywiEAoMSWRlbWl4T3JnTVNQEAEqzgcKRAogDCvVIZKf3BjWSglYHs9hpmIYFoivU2Ny/ikuMz55VmkSICGZg9JMY+E2eGDpXMowMIW/q6qr5f6ruBHewxDeonVhEkQKICqW2eFFYce3rL+W+vu4YEW83SBWpBUae3ZXdtpytC/BEiAO7VqjyDybt5G4yIkRnZt0MaizCkXEIrEcTF5GhEdYdBpECiAAboYv9DUo8rfC2Nn9tF2YoyVh35YrYX60mjfMS/ugGxIgBRL1EIkE2PWm/prA1S4hAwrmHRnnF2FyrKkK1HJVU3siIBB/5FVWOQnly0LQYeW8Xhm1Y1zNFU4YAZ3n4Pn+W1B6KiAlZdXIel/1SQMDjOwUTKJCenIa8gl2Tyg2p3jWtNX/AjIgELICUsvWTp74zgOVGpCdVpYllbLF19jTPup0sDEQHRc6IB5z48Dokk3HGmfT555wnvwLELoid1lmmRlOmlhSfo3rQiANhc+0gpNXAS2XVoZ+fWcROim/cCK1b/f25/YJp9+42EogEsE+x5fHs+4n4+ve8Lnz8UBdvdEYTdfHNQfjETofOr1SIA6Fc43xTun9EYUKuDIpPdVqLli9BilsmHmhhyKv2U6HUiAUYOPpVIbr4PwDpEd5QdET9bu+qIZTx1SQEsq3Ky8TTlogIDYq3UNHAY1tdC0+fiVUM5c2WWBcehqcavMlTDuQxWpiRAogAcJPi5AA8Q/Xzzd1fGwlheyGb6v/xKX4pYpAE+2wMy0SIA3Wqd3xAoXZhK/oF+3O3W/leoffTLMmqXOnvvsWgFGzaiAl/5bWO/zuNCcqEUqLQC7L8ySTQSg5e7w9jP+IZsNc23KIAQogGY6Tk5INSDpyYL+3MftdJfGqSTM1qecSl+SFt67zEsISIBgA3u8SHx52QmoAZl5cRHlnQyLU917a3UbevVzZkvbtGiAJBonQWF/wdeyema1pDDOVvEsxM3CzjvNVrNrc0SKXWyIgEshepduMbetKq3GAjctAj+PR52kMQ9N7TObMAWb6fap6ZzBlAjEA3YJerRXWIkSqyNJj9GWNV1iGDgRaR9F8uSP1ogYdmY0ORg8zIgwntDg41rS7E+JSAjAQowDJ4JD5jNx2O4AFd8JqBQd0BcrEfw+QdlFVhqiHG8wjX67/cT369jPpZtajxdCKAQCSAWgKRAogB3S14S2mIIQOadbFeGXNFOJjxTYcMdMDgO0wxm6M6IISIBp8N5qfXX941p2f4jklZ9lOe07TaD/XvPq9MgljWUikEiAKUJC0MZGIHvJf9lkIHgaZlS00Qi5S30vWvId1rLeqeg=="
      },
      "Type": "TOK",
      "DecimalQuantity": "10"
    }
  ]
}
```

Alice can now transfer some of her tokens to other parties. For example:

```shell 
./fungible view -c ./testdata/fsc/nodes/alice/client-config.yaml -f transfer -i "{\"TokenType\":\"TOK\", \"Quantity\":6, \"Recipient\":\"bob\"}"
```

The above command instructs Alice's node to perform a transfer of 6 units of tokens `TOK` to `bob`.
If everything is successful, you will see something like the following:

```shell
"26e1546970b96299680040b3409cfcd486854cbbf13e1e46d272cf85127b271c"
```

The above is the transaction id of the transaction that transferred the tokens.

Now, we check again Alice and Bob's wallets to see if they are up-to-date.

Alice:

```shell
./fungible view -c ./testdata/fsc/nodes/alice/client-config.yaml -f unspent -i "{\"TokenType\":\"TOK\"}"
```

You can expect to see an output like this (beautified):

```shell
{
  "tokens": [
    {
      "Id": {
        "tx_id": "26e1546970b96299680040b3409cfcd486854cbbf13e1e46d272cf85127b271c",
        "index": 1
      },
      "Owner": {
        "raw": "MIIEmxMCc2kEggSTCgxJZGVtaXhPcmdNU1ASggkKIBcbZLS9FhXl09IqczHADl22FHW6oSFxpFVy1lzjnvKwEiAAE7YWWVG3xkUt4Hj37t3maDh35u3GQSgS4Xe7p0seUxpZCgxJZGVtaXhPcmdNU1ASJ2RlZmF1bHQtdGVzdGNoYW5uZWwtZGVmYXVsdC5leGFtcGxlLmNvbRogAvYocoRRzxnU/3a5uT0Lo/0S+q6nBi5/kLf96ay1gywiEAoMSWRlbWl4T3JnTVNQEAEqzgcKRAogGi67Lnc5ponluEm4YB/ZtnYDNOn9rRwgirw7KwDAuvASICLCJCB3zoQYnt76MTVjKwrEPTW9ShADObpX8RGGo1WmEkQKIA3npyjuwVfxyCJ8RZCl4Q3lZ7BbRZW6skkVV4uJ9LDaEiAHO2Vh4HSjO1ct44w5Evp1YdjGC0l2N+EoGGg1jjRKzhpECiAl+Wmp34OubzO64TyntSBoAjCPCc1gfBXB2oyQMnuwMxIgE5W0f0qayux2/lJmbs4IDSgVxwHdwFtfDcadrMoqHBAiIAVigpyA1/M9bEZX9ihbgKRFHVlGq5gFrTUF/7JMc39wKiAdp6WB7NGy+kpaBNMpquYGhNijjmbkArgLzLxx0jdefDIgHqYoHP33vdOi+/Hwec1nIpC3b5KMto4oVQlcjwKIfRs6ICoNL+Sm/a6BqOAoju1UoCCxHnVkROk9HCl7UQR42lIuQiAmJEQkJAHGriYhh+6mcrsf/uH6p3JSSzBGDOSuw1qNGEogGnFtcE6Hde4P7knQppY9148A7WB03x22DhETnPEoM4xSIB/bPQ1oL1DZURI34B7HeRNwueR/kTti2S7FYXRjLq15UiAprNE30EAxMFQ9juO5FdSpWrOi1Wn2yHrG30G+5+9x8logFYxXD3llanJ4SCKAx0dJ9bFsQPtplbE2Ad3UdOD99qliRAogFxtktL0WFeXT0ipzMcAOXbYUdbqhIXGkVXLWXOOe8rASIAATthZZUbfGRS3gePfu3eZoOHfm7cZBKBLhd7unSx5TaiAFPZYpfR8yNh2lfOonbXZKWDu6A9nn1k93XdTKbBE2MnKIAQogGY6Tk5INSDpyYL+3MftdJfGqSTM1qecSl+SFt67zEsISIBgA3u8SHx52QmoAZl5cRHlnQyLU917a3UbevVzZkvbtGiAJBonQWF/wdeyema1pDDOVvEsxM3CzjvNVrNrc0SKXWyIgEshepduMbetKq3GAjctAj+PR52kMQ9N7TObMAWb6fap6ZzBlAjEA3YJerRXWIkSqyNJj9GWNV1iGDgRaR9F8uSP1ogYdmY0ORg8zIgwntDg41rS7E+JSAjAQowDJ4JD5jNx2O4AFd8JqBQd0BcrEfw+QdlFVhqiHG8wjX67/cT369jPpZtajxdCKAQCSAWgKRAogKthwRsZE5VxE3m6IHJ5ifN0E9fWuAPoOhnOlkhsFvPQSIAbTbuvi6fuccEzywmq6nIO8zgFW1YbkFevHggIDeZjqEiAdSXW+RnXJkI/uN7QPlcrmvAxmF4qH7v9wAy0QKTceaA=="
      },
      "Type": "TOK",
      "DecimalQuantity": "4"
    }
  ]
}
```

Then, Bob:

```shell
./fungible view -c ./testdata/fsc/nodes/bob/client-config.yaml -f unspent -i "{\"TokenType\":\"TOK\"}"
```

You can expect to see an output like this (beautified):

```shell
{
  "tokens": [
    {
      "Id": {
        "tx_id": "26e1546970b96299680040b3409cfcd486854cbbf13e1e46d272cf85127b271c"
      },
      "Owner": {
        "raw": "MIIEmxMCc2kEggSTCgxJZGVtaXhPcmdNU1ASggkKICUQcEN/52Caweb0KpnfO/ThAFKHOv4a3c8Tdmyd4nXdEiAedZFC0VgWtS41eoFPbyW5HCt32JecChY7VSkqSj58bhpZCgxJZGVtaXhPcmdNU1ASJ2RlZmF1bHQtdGVzdGNoYW5uZWwtZGVmYXVsdC5leGFtcGxlLmNvbRogAvYocoRRzxnU/3a5uT0Lo/0S+q6nBi5/kLf96ay1gywiEAoMSWRlbWl4T3JnTVNQEAEqzgcKRAogCPq27TzRNd5juExU3tG8Gz093rrctGSDw8j70UtVHKoSIBoTzD15AU+k3dK+mpEOh15tmreZUDFPjTGpWJnoT5SMEkQKIAzXax75L8lZifdm2Mw3mOZP9BOl0iTbHC+sJ9u6tgcDEiAiMwrR2zsUowm2O4kUj6wQBVoWRfKahQgLIdXoQBjlnxpECiAQOAyPj4KAG4wgHxj1ZzwlVu4V2VOCmaU+ORTMBucINxIgIlcGWlHAMn8iLVHIj64B0IvGCayFtSEb5oicruv2llciIBjCsbLVbhQm0EABn8fqycPmiOnJwBRkLEIcVoE6iJ8GKiAYAEyH+ZCg6nrZvLbM8IgYdvyPYHDFD4WGfcfMOyQcuzIgHRSPOHxlcneVnn+IY28Rrd6qsfTpNPPrfk12jAwo5BA6IBNB3AoffpT1iDS05wEKAV1khK3r+6aa/1aOZWM8RWkhQiAq2gHduDKhSbTO3lIw0Nv+u7popduVFHBub/yBw1yN+kogAhg82iwgvf+ymaRNO8XXAq/XFzt6ZHJKsAD0q/Uz6cdSIBcOaU4ttkLHTJT9WrKLoA1L++1fB9XEpB1IC2q4w/cjUiAik8nVmX7OKKR8vF6IovWKVfn4EB1J2a3japRFpHYIqVogK+PzUp1i3gp4QKycTbvMGz7kYRBnan012GUEA+UzfxhiRAogJRBwQ3/nYJrB5vQqmd879OEAUoc6/hrdzxN2bJ3idd0SIB51kULRWBa1LjV6gU9vJbkcK3fYl5wKFjtVKSpKPnxuaiAgqHc732XZd7yfKCpnk9XH4toLpFu/odPT5LrZSgUeWnKIAQogGY6Tk5INSDpyYL+3MftdJfGqSTM1qecSl+SFt67zEsISIBgA3u8SHx52QmoAZl5cRHlnQyLU917a3UbevVzZkvbtGiAJBonQWF/wdeyema1pDDOVvEsxM3CzjvNVrNrc0SKXWyIgEshepduMbetKq3GAjctAj+PR52kMQ9N7TObMAWb6fap6ZzBlAjEAyA9+362MWyeF8QllJYEba/zs0x2wTF60ZtteERimKRQQLVdJzBpx9MgyonUNIy4CAjAdyMpi+0ki6re3HW+uma/qN2qr2WMVJA4PgIhdBmR+b8zb6+NyvNJ/t8waF3ZqUhqKAQCSAWgKRAogG9kaVKu1CdP8L+1n0juQyQJaVjn5fXpKMjHymrpAlrsSIAbB6cz91ElZIYuJJGCCIBHYJiEkvrWAudqTPfjF9iuSEiAkoA9Pt5cG3xyzGyhN37nGRo33VWXkYcQYaHqoXwin6w=="
      },
      "Type": "TOK",
      "DecimalQuantity": "6"
    }
  ]
}
```

Let us try something more complex: A swap. 
Let us consider the following scenario: Alice wants to swap 1 unit of TOK tokes with 1 unit of KOT tokens owned by Charlie.

Because KOT tokens don't exist yet, let us issue them first.
The following command instructs the issuer to issue KOT tokens to Charlie.

```shell
./fungible view -c ./testdata/fsc/nodes/issuer/client-config.yaml -f issue -i "{\"TokenType\":\"KOT\", \"Quantity\":10, \"Recipient\":\"charlie\"}"
```

This is transaction id of the transaction that issues KOT tokens
```shell
"ca195ab8072f12d6e0ed78d977404668e0523dbd45b72991cb2cd853de58e5e8"
```

Let us query Charlie's wallet

````shell
./fungible view -c ./testdata/fsc/nodes/charlie/client-config.yaml -f unspent -i "{\"TokenType\":\"KOT\"}"
````

You can expect to see an output like this (beautified):


```shell
{
  "tokens": [
    {
      "Id": {
        "tx_id": "ca195ab8072f12d6e0ed78d977404668e0523dbd45b72991cb2cd853de58e5e8"
      },
      "Owner": {
        "raw": "MIIEnBMCc2kEggSUCgxJZGVtaXhPcmdNU1ASgwkKIAdT9TwkNH+V6BQ+/lI5b4Am76lF0Vk8ZpQEYTWZ6cGBEiAV8SYHdCviYcgBXzePH4rckaBMIXR0T3jbTn/rORABtBpZCgxJZGVtaXhPcmdNU1ASJ2RlZmF1bHQtdGVzdGNoYW5uZWwtZGVmYXVsdC5leGFtcGxlLmNvbRogAvYocoRRzxnU/3a5uT0Lo/0S+q6nBi5/kLf96ay1gywiEAoMSWRlbWl4T3JnTVNQEAEqzwcKRAogFFpI5IVoV2oIhJnl477WAWeTYiDrHnr/V6QodyFyycESIBxPmg2eZO6wtiSt1nbmpo1yZIxIUNyiKy8XpsYc5cJcEkQKICT9MLpXc0pfYY6KGsvGcvl16gPrg+BdRl7WV0CgUmnoEiAcGBrN3/5wX+eWG31xX69gzM+c9Nk1imtyLG4qoqF25xpECiAjslZCXcCHHUaPv00mVW4RKPwRxMI9gZUSCyWc2Q8byxIgBk7LJ8vpIn85VpE4ZVPgUypb4P3SjPhXln2u07fPNrIiIBOxJvXSouyqljfNJb8S5ThNFO7U46N216c4f9sAjfP0KiAkozy8lbwlKhuPiPA0jqqLGAt0X1mxQKy8ZgTnZ43HRDIgAozvYrwYm7cYuU085+C7uvP7KTMHO9luNv8n1vHzf0I6IBM2SDoCyhitQCDrkki0e+HvbNjQ1EWVy7Dbd74G8EgBQiAeJV6NSR4/l9gfFD3zxpeBNtPrgSYqPSN19I19XcsmfEogK8NJLi+4gaRdALNAOsQddfynlKHlQAYV8YOXuaIInzNSIBVfum33qgBFZYCWFKLRN3/WJj1yg7d802tHZewoy4glUiAb8O+SUBKtG6SQxcFijrRUw6D6jWBg9xj0PlSpB/JZLFogI1U7eydWvNENNmIQlgaIs+5oFwz2gyfcaYOdR6clbPtiRAogB1P1PCQ0f5XoFD7+UjlvgCbvqUXRWTxmlARhNZnpwYESIBXxJgd0K+JhyAFfN48fityRoEwhdHRPeNtOf+s5EAG0aiAglY3WmDBF0eg5FHCVfVCt3wD6NSkyFUzFzcbesfYNGHKIAQogGY6Tk5INSDpyYL+3MftdJfGqSTM1qecSl+SFt67zEsISIBgA3u8SHx52QmoAZl5cRHlnQyLU917a3UbevVzZkvbtGiAJBonQWF/wdeyema1pDDOVvEsxM3CzjvNVrNrc0SKXWyIgEshepduMbetKq3GAjctAj+PR52kMQ9N7TObMAWb6fap6aDBmAjEAzjgDxfFwSzA2VLKaQofXmbQCsPM3CBvzU64G/Yem38fybHlkSnnyv/zGRv275Fj7AjEAldpJLRUrK6TPxxOhMZsxeklROHRM+RZOe+9B7UBuojYA8RJXRRALE2rqB+EiyCXiigEAkgFoCkQKIBWjarOnKvfmEIXEfpIymPYlwoa5GwamwkQLU+dDIHLMEiAOfw/mXGo3WcuamtnWab6JRu6NjgwvQWSrkgwtLu/rOhIgAHCrCKb8BvO46JAJkgZGBy+44K1ljJD6VMER0O4/H4k="
      },
      "Type": "KOT",
      "DecimalQuantity": "10"
    }
  ]
}
```

Now, we are ready to orchestrate the swap.
The above command instructs Alice's node to start the swap we described above whose business process was described in the
previous sections.

```shell
./fungible view -c ./testdata/fsc/nodes/alice/client-config.yaml -f swap -i "{\"FromType\":\"TOK\", \"FromQuantity\":1,\"ToType\":\"KOT\", \"ToQuantity\":1, \"To\":\"charlie\"}"
```

If everything is successfully, you can expect to see the transaction id of the transaction that performed the swap.

```shell
"790098ae67dc5157c7d679b8def39983adb6e9d7070209f5ab12938d45e1630d"
```

We can now query the wallets of Alice and Charlie to confirm the swap happened.

Alice:

```shell
./fungible view -c ./testdata/fsc/nodes/alice/client-config.yaml -f unspent -i "{\"TokenType\":\"\"}"
```
 
You can expect to see that Alice has 3 units of TOK tokens, and 1 unit of KOT tokens.

```shell
{
  "tokens": [
    {
      "Id": {
        "tx_id": "790098ae67dc5157c7d679b8def39983adb6e9d7070209f5ab12938d45e1630d",
        "index": 1
      },
      "Owner": {
        "raw": "MIIEmxMCc2kEggSTCgxJZGVtaXhPcmdNU1ASggkKICJZpYJkYEt1+0bTwp9yIsbIwT+9uTyDkCvaBEZQINZMEiAINuE8/cVzKaQUKcbUX31hMkKuDAk9AQ7780C0ZVXRTRpZCgxJZGVtaXhPcmdNU1ASJ2RlZmF1bHQtdGVzdGNoYW5uZWwtZGVmYXVsdC5leGFtcGxlLmNvbRogAvYocoRRzxnU/3a5uT0Lo/0S+q6nBi5/kLf96ay1gywiEAoMSWRlbWl4T3JnTVNQEAEqzgcKRAogLttMr+WJ06CTka5Kel024/b0IQTMzgoRxq3bs6aTZ18SIA2DpE4uY2foJ8GkPy1z6wmGCPjMGYJjnaZ+96crIwqNEkQKICrCRwCQIDQong3+z2emWZJC6MGnEO9DaYb3+fjKmTfSEiAtCGFNA8z1rhgQjxdVUFNvrk3f/gb8H/E9e+tOu888pxpECiAoBB0GmanqV+SEkP/u3sJiI/LllZ13gtiiZzfqO9JKoBIgF90F29Rc3fXp/fKRjIY7Dcg+bWN4EV8B+tw1ft9196QiIAjDN69DMy27cR5ykFsYsRJey0/xpSqtZwdEpy71TsGYKiAPR/JgslMlgkzGuhp+ZH3PL7Kapl4VzdexwfL8rZDppDIgI02k6BlrsT2Vh2nIXH7NzZGhqs91Xgdlzc8CIjjzskM6IBBqjy4rKIh8MrWHyyXZ4UNWUC4ddCYCJsbeO8dqt+RlQiALPa15oWp++3RPMM5GMX5yUTHCMjH+BGzV8jIBjmJhlkogEjsFfo7PPEV/IAWDNDirnEiXLhhXNXWrp+75/IHzyAlSICMml9G4B2Ci6dvkCU60tXBjsjMOhnWrOElU1c/AG6g5UiARiUDjF9zP5Awdm1hQN2rLCyGqbEOhvxO6HAN9sw+p7FogCCnGnwolxd6rwE6onuOgmm6X4L9mSxmI21q2Of8Ki+RiRAogIlmlgmRgS3X7RtPCn3IixsjBP725PIOQK9oERlAg1kwSIAg24Tz9xXMppBQpxtRffWEyQq4MCT0BDvvzQLRlVdFNaiAFcdKLBbkxlV+8B7Db3LwdHBKPJ58HiMgMMkv1NA3xKXKIAQogGY6Tk5INSDpyYL+3MftdJfGqSTM1qecSl+SFt67zEsISIBgA3u8SHx52QmoAZl5cRHlnQyLU917a3UbevVzZkvbtGiAJBonQWF/wdeyema1pDDOVvEsxM3CzjvNVrNrc0SKXWyIgEshepduMbetKq3GAjctAj+PR52kMQ9N7TObMAWb6fap6ZzBlAjEA3YJerRXWIkSqyNJj9GWNV1iGDgRaR9F8uSP1ogYdmY0ORg8zIgwntDg41rS7E+JSAjAQowDJ4JD5jNx2O4AFd8JqBQd0BcrEfw+QdlFVhqiHG8wjX67/cT369jPpZtajxdCKAQCSAWgKRAogFllIt96ke1j9jVuzhdAQz+KRlOtM6SKFR6rwVukkOakSICKsh2fdSfpzP0r7tNUfoOCTw9KQKkq11WwHYN6RmtlsEiAs7EYkgGcoKDHky5yXN7qQjr/FFQfEKkG8KgEbAWy5Og=="
      },
      "Type": "TOK",
      "DecimalQuantity": "3"
    },
    {
      "Id": {
        "tx_id": "790098ae67dc5157c7d679b8def39983adb6e9d7070209f5ab12938d45e1630d",
        "index": 2
      },
      "Owner": {
        "raw": "MIIEmxMCc2kEggSTCgxJZGVtaXhPcmdNU1ASggkKIAVafcxfRK0H0QWSQM8oy2p+CIWRfDxHGxIVdQwMiZvUEiAN310C96rxXnLZ0xbR1TFj7uWYyXwgXJYy6DR4vOgbTRpZCgxJZGVtaXhPcmdNU1ASJ2RlZmF1bHQtdGVzdGNoYW5uZWwtZGVmYXVsdC5leGFtcGxlLmNvbRogAvYocoRRzxnU/3a5uT0Lo/0S+q6nBi5/kLf96ay1gywiEAoMSWRlbWl4T3JnTVNQEAEqzgcKRAogCuypu8bCCgJz8jeQuLeMfAUpt57B8r91GKuBxy0EDxUSIASP2yYOOTaLvle/TfM79FCsWYnmsjjy4OokZ2MfwbsREkQKICTppfd2awQh4bXRuHuSYyaoBKVDl/KOpw5RTeYZ6eebEiAIruuLKQwpSo6iHrV8ui/JiKPPEX3/8AAvvJJJlc6lzxpECiAnDCyWWnk3T7r515qQM+Jp0gKYMSSJ4W11+r7lEtrQsBIgDKnAphIt06U04MfGR8F7YRf1mVIi7Z+8aBV3VIK+WxIiIBjPazywKo/XaxorqU2Mnyz138W4zE7lfyYaF7b39ahtKiADlNiVapRCF7NiFjEE5yGRpRZFiRUYXwzw5rF3R4i05zIgJd2JbWMXIaIILplk3Ui9PiAMXYI6Q2uC/TVfU4j6H/M6ICMegnZ40/geKGj3+NAQO0ONkWK+GpyijyobmznG1uD0QiAZDctUyhkwccsEUSLkOdR6aR9yTAzcfalbBv5oItTuQkogCN+dLBPNx8IUPfwes3I2LvupSen6jCZhygLeTvREEy1SICJgsck+7Kdwt03svl5DPg95B9ST8MkrTCLWlGTkghHyUiATlAEj7R5uZy7wKqqJnoJWR9tPPaVdVsz3O3GuI2dXhFogCj9aW1AE+qVx+Q5G7/bvaMlaJaWjidMdeHU/CfKV5kZiRAogBVp9zF9ErQfRBZJAzyjLan4IhZF8PEcbEhV1DAyJm9QSIA3fXQL3qvFectnTFtHVMWPu5ZjJfCBcljLoNHi86BtNaiAQDx6X9QOy6aj9LV0FHoxdfLZrnelqR4BwGjNmdgEhG3KIAQogGY6Tk5INSDpyYL+3MftdJfGqSTM1qecSl+SFt67zEsISIBgA3u8SHx52QmoAZl5cRHlnQyLU917a3UbevVzZkvbtGiAJBonQWF/wdeyema1pDDOVvEsxM3CzjvNVrNrc0SKXWyIgEshepduMbetKq3GAjctAj+PR52kMQ9N7TObMAWb6fap6ZzBlAjEA3YJerRXWIkSqyNJj9GWNV1iGDgRaR9F8uSP1ogYdmY0ORg8zIgwntDg41rS7E+JSAjAQowDJ4JD5jNx2O4AFd8JqBQd0BcrEfw+QdlFVhqiHG8wjX67/cT369jPpZtajxdCKAQCSAWgKRAogFLurOOSmbEEk7yNEwRX9ghWk7IbWDnA8V9kLB6gEWyMSIBOjwYbM9HA30fr9usDn16m5chNpyDEgi6ktlhfRTQ4REiAwXYEeOISdw+4YSSVS9PeqZK5f5EqsV/6xwALU20XJ5w=="
      },
      "Type": "KOT",
      "DecimalQuantity": "1"
    }
  ]
}
```

Charlie:

```shell
./fungible view -c ./testdata/fsc/nodes/charlie/client-config.yaml -f unspent -i "{\"TokenType\":\"\"}"
```

You can expect to see that Charlie has 1 unit of TOK tokens and still 9 units of KOT tokens.

```shell
{
  "tokens": [
    {
      "Id": {
        "tx_id": "790098ae67dc5157c7d679b8def39983adb6e9d7070209f5ab12938d45e1630d"
      },
      "Owner": {
        "raw": "MIIEnBMCc2kEggSUCgxJZGVtaXhPcmdNU1ASgwkKICGsY7CO3ltYugDI2GImFCD5WOCJq7HDI0I+Ki41nM/XEiAVb1ZOtv+Aod7xNOdpeTudN88+7sRSDTmx5yW4VukgJBpZCgxJZGVtaXhPcmdNU1ASJ2RlZmF1bHQtdGVzdGNoYW5uZWwtZGVmYXVsdC5leGFtcGxlLmNvbRogAvYocoRRzxnU/3a5uT0Lo/0S+q6nBi5/kLf96ay1gywiEAoMSWRlbWl4T3JnTVNQEAEqzwcKRAogGXnVxqgPuf0/pbEjKQV01cCFrWF9mCJBubPOSnpmrUgSIBr4WDNTvTmcuLYXDJc+5PVZgojMw1edFlxKdo/hJBAvEkQKIAoBWg7DYx1SrHiJu3okRrMRYidR39v9R/visJwLmifjEiAVQuPsDdTupMpuM65dT+mD+simDhQf/jmgWlFrhmzsVBpECiAb4Jz/xjedjxp4XvFz5I9xZDGN952T3yMv6viybDqo7hIgA8LebWf/Kr4o2lPHDS04m/hkviuKhfLZcmj/OpLDkuAiICWZHBevFDIGmSJ+3ibmH55zccGGNTFTPVAZ28JxSKASKiAZzuai4Qz2PxdEcALgVSgWox3Nslxi9uK6LkJ+lb61JjIgI3MPK5Abo+rU+zOlsVyQeQH4UyqLqWlidUGdAd6Ipf06IBt5IxBUiO29clqMNjFtjtN9ZcmY110CcADI7yu2SRbaQiAYFBpAre8HUOy/42h59HlwXdXlFFW6IvGBznpKkNsqgkogKxVRAtjKZkrpYFlgRvPoUgs/Rb2zhMwWrPO8iV0X2CRSICsmtLvRi+jZppqY5lWenFBVQ1rW9p9n4nrWngVJ92h9UiAYMdYcyGCOYDdckx8aEgV7Owo1uAcB8Gcpti1Uej0vwFogHYpzOrGr8h9+x/2Pz/GT6oDYOjosCY2tmeWdyzmLy3piRAogIaxjsI7eW1i6AMjYYiYUIPlY4ImrscMjQj4qLjWcz9cSIBVvVk62/4Ch3vE052l5O503zz7uxFINObHnJbhW6SAkaiABlLc54f1rvWjm+CpY+SJCJMDaSsHuDrpF+8+5pwT/VnKIAQogGY6Tk5INSDpyYL+3MftdJfGqSTM1qecSl+SFt67zEsISIBgA3u8SHx52QmoAZl5cRHlnQyLU917a3UbevVzZkvbtGiAJBonQWF/wdeyema1pDDOVvEsxM3CzjvNVrNrc0SKXWyIgEshepduMbetKq3GAjctAj+PR52kMQ9N7TObMAWb6fap6aDBmAjEAzjgDxfFwSzA2VLKaQofXmbQCsPM3CBvzU64G/Yem38fybHlkSnnyv/zGRv275Fj7AjEAldpJLRUrK6TPxxOhMZsxeklROHRM+RZOe+9B7UBuojYA8RJXRRALE2rqB+EiyCXiigEAkgFoCkQKICFpz7fe2B1/KuLDr2BSRVbABRQaXchNpuNJWT8iHXMuEiAgBnZ44Ww3M4VVUiZ59uqtFGLwJ/D3Da4CFyaGylWblhIgEeqORny6JkziE31kvUXv72SXOUlNZqAmQKMZaMF0NqI="
      },
      "Type": "TOK",
      "DecimalQuantity": "1"
    },
    {
      "Id": {
        "tx_id": "790098ae67dc5157c7d679b8def39983adb6e9d7070209f5ab12938d45e1630d",
        "index": 3
      },
      "Owner": {
        "raw": "MIIEnBMCc2kEggSUCgxJZGVtaXhPcmdNU1ASgwkKICVPsqioUyWm9q6xizxbHfgBSSUkg0s0b3hUhe4zbQQYEiAZ8BHcI3BBs0qtGX49E9lngYv4b0Qb4EhCnYAi2WGxqhpZCgxJZGVtaXhPcmdNU1ASJ2RlZmF1bHQtdGVzdGNoYW5uZWwtZGVmYXVsdC5leGFtcGxlLmNvbRogAvYocoRRzxnU/3a5uT0Lo/0S+q6nBi5/kLf96ay1gywiEAoMSWRlbWl4T3JnTVNQEAEqzwcKRAogBTwzc7U816TnMRwVxY8NFWV4IQ99jPAIG9Qnsn+ZPRsSIAdnZwExRnQZV5TiKRaqPmWpPJN/Nmwz6bg2vW0Dl28WEkQKIAE551T9rDBCUBwLQ8iEHneHiFTyDN1BI5bYKJ2E62c1EiAR6KLGNrx48iW5LZQF0QUWURJoxPuSmw3cYrXRsHKnfBpECiArBYSRzJqPTiPLCc/4XUeFfEPSnfQxNgPEk0rWWGpQwxIgDo1QCcOIjH4BW4nM67UQU6LmaPIrZm5QPMkRm0zNqbEiICtIljP4XkWQWuhzzUWymOtYf+jEvxO5q0DjFDaXfNY7KiARo8TWfG8c8T16pOXOLcLPWAzd8i5v3STWyGV7f2NJSzIgFJGYzwy7WWuG/I+BpOgaI6W/W64naQNPy4LTFb8GHQ06IDBC9vD8nMII4b9z6Co76tPbYfsUPg1HGLBXZwtO59ySQiAXkDX1J4Ee6TMJ+XCTiqNGoBD/78zkKA4nkMfBnyuO6kogDxN/pIlqDccyiCFb3GYkIDAU1nAZBhWy3baIZA3uNzpSIAE/U5CQi0+2P3rxaBtMbuHztpyWZXmRxkwE5v8cN8h0UiAiEbRhFC3NzbqOI+F3WTRuRAaKILU4nJQLfnVn3TcgGlogLDfRJ099L1Z0tNnrymmAaOzR156wg8n85RygbfEAe4piRAogJU+yqKhTJab2rrGLPFsd+AFJJSSDSzRveFSF7jNtBBgSIBnwEdwjcEGzSq0Zfj0T2WeBi/hvRBvgSEKdgCLZYbGqaiAXF4wRyDbFe9/rAgUYfM49iI/HU6hb9WzM1syJXeJ1knKIAQogGY6Tk5INSDpyYL+3MftdJfGqSTM1qecSl+SFt67zEsISIBgA3u8SHx52QmoAZl5cRHlnQyLU917a3UbevVzZkvbtGiAJBonQWF/wdeyema1pDDOVvEsxM3CzjvNVrNrc0SKXWyIgEshepduMbetKq3GAjctAj+PR52kMQ9N7TObMAWb6fap6aDBmAjEAzjgDxfFwSzA2VLKaQofXmbQCsPM3CBvzU64G/Yem38fybHlkSnnyv/zGRv275Fj7AjEAldpJLRUrK6TPxxOhMZsxeklROHRM+RZOe+9B7UBuojYA8RJXRRALE2rqB+EiyCXiigEAkgFoCkQKICkeIT/6W6IQpr9QNROBSR69w5ynLg01vSqMZrqGP+oGEiANCVz7nY94sX1MxYATg0BWVz/j69lP2R06i7V5LFY7OBIgCMItl/7wSRXVWxYszLX7UgsKdBgDV6Bo7KTAnszIHmQ="
      },
      "Type": "KOT",
      "DecimalQuantity": "9"
    }
  ]
}
```