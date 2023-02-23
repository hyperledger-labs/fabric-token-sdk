# Token SDK, Non Fungible Tokens, The Basics

In this Section, we will see examples of how to perform basic token operations 
like `issue` and `transfer` on `non-fungible tokens` (NFT, for short).

We will consider the following business parties:
- `Issuer`: The entity that creates/mints/issues the tokens.
- `Alice` and `Bob`: Each of these parties is a `NFT` holder.
- `Auditor`: The entity that is auditing the token transactions.

The NFT we use will model a `house` with an address and a valuation. 

Each party is running a Smart Fabric Client node with the Token SDK enabled.
The parties are connected in a peer-to-peer network that is established and maintained by the nodes.

Let us then describe each token operation with examples:

## Issuance

Issuance is a business interactive protocol among two parties: an `issuer` 
and a `recipient` that will become the owner of the freshly created NFT.

Here is an example of a `view` representing the issuer's operations in the `issuance process`:  
This view is executed by the Issuer's FSC node.

```go
// IssueHouse contains the input information to issue a token
type IssueHouse struct {
	// IssuerWallet is the issuer's wallet to use
	IssuerWallet string
	// Recipient is an identifier of the recipient identity
	Recipient string
	// Address is the address of the house to issue
	Address string
	// Valuation is the valuation of the house to issue
	Valuation uint64
}

type IssueHouseView struct {
	*IssueHouse
}

func (p *IssueHouseView) Call(context view.Context) (interface{}, error) {
	// As a first step operation, the issuer contacts the recipient's FSC node
	// to ask for the identity to use to assign ownership of the freshly created token.
	// Notice that, this step would not be required if the issuer knew already which
	// identity the recipient wants to use.
	recipient, err := nftcc.RequestRecipientIdentity(context, view2.GetIdentityProvider(context).Identity(p.Recipient))
	assert.NoError(err, "failed getting recipient identity")

	// At this point, the issuer is ready to prepare the token transaction.
	// The issuer creates an anonymous transaction (this means that the result Fabric transaction will be signed using idemix),
	// and specify the auditor that must be contacted to approve the operation
	tx, err := nftcc.NewAnonymousTransaction(
		context,
		nftcc.WithAuditor(
			view2.GetIdentityProvider(context).Identity("auditor"), // Retrieve the auditor's FSC node identity
		),
	)
	assert.NoError(err, "failed creating issue transaction")

	// The issuer adds a new issue operation to the transaction following the instruction received
	wallet := nftcc.GetIssuerWallet(context, p.IssuerWallet)
	assert.NotNil(wallet, "issuer wallet [%s] not found", p.IssuerWallet)
	h := &House{
		Address:   p.Address,
		Valuation: p.Valuation,
	}
	// The issuer enforce uniqueness of the token by computing a unique identifier for the passed house.
	uniqueID, err := uniqueness.GetService(context).ComputeID(h.Address)
	assert.NoError(err, "failed computing unique ID")

	err = tx.Issue(wallet, h, recipient, nftcc.WithUniqueID(uniqueID))
	assert.NoError(err, "failed adding new issued token")

	// The issuer is ready to collect all the required signatures.
	// In this case, the issuer's and the auditor's signatures.
	// Invoke the Token Chaincode to collect endorsements on the Token Request and prepare the relative Fabric transaction.
	// This is all done in one shot running the following view.
	// Before completing, all recipients receive the approved Fabric transaction.
	// Depending on the token driver implementation, the recipient's signature might or might not be needed to make
	// the token transaction valid.
	_, err = context.RunView(nftcc.NewCollectEndorsementsView(tx))
	assert.NoError(err, "failed to sign issue transaction")

	// Last but not least, the issuer sends the transaction for ordering and waits for transaction finality.
	_, err = context.RunView(nftcc.NewOrderingAndFinalityView(tx))
	assert.NoError(err, "failed to commit issue transaction")

	return h.LinearID, nil
}
```

The important point to highlight is that the issuer must ensure that the NFT is unique in the system.
There are many ways to do that. The simplest one is to compute a salted hash of the data structure (or part of it) one
wants to convert to an NFT. This is the approach followed by the `uniqueness` package.

Here is the `view` representing the recipient's operations, instead.  
This view is execute by the recipient's FSC node upon a message received from the issuer.

```go
type AcceptIssuedHouseView struct{}

func (a *AcceptIssuedHouseView) Call(context view.Context) (interface{}, error) {
	// The recipient of a token (issued or transfer) responds, as first operation,
	// to a request for a recipient.
	// The recipient can do that by using the following code.
	// The recipient identity will be taken from the default wallet (ttx.MyWallet(context)), if not otherwise specified.
	id, err := nftcc.RespondRequestRecipientIdentity(context)
	assert.NoError(err, "failed to respond to identity request")

	// At some point, the recipient receives the token transaction that in the mean time has been assembled
	tx, err := nftcc.ReceiveTransaction(context)
	assert.NoError(err, "failed to receive tokens")

	// The recipient can perform any check on the transaction as required by the business process
	// In particular, here, the recipient checks that the transaction contains one output that names the recipient.
	// (The recipient is receiving something)
	outputs, err := tx.Outputs()
	assert.NoError(err, "failed getting outputs")
	assert.NoError(outputs.Validate(), "failed validating outputs")
	assert.True(outputs.Count() == 1, "the transaction must contain one output")
	assert.True(outputs.ByRecipient(id).Count() == 1, "the transaction must contain one output that names the recipient")
	house := &House{}
	assert.NoError(outputs.StateAt(0, house), "failed to get house state")
	assert.NotEmpty(house.LinearID, "the house must have a linear ID")
	assert.True(house.Valuation > 0, "the house must have a valuation")
	assert.NotEmpty(house.Address, "the house must have an address")

	// If everything is fine, the recipient accepts and sends back her signature.
	// Notice that, a signature from the recipient might or might not be required to make the transaction valid.
	// This depends on the driver implementation.
	_, err = context.RunView(nftcc.NewAcceptView(tx))
	assert.NoError(err, "failed to accept new tokens")

	// Before completing, the recipient waits for finality of the transaction
	_, err = context.RunView(nftcc.NewFinalityView(tx))
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

Transfer is a business interactive protocol among at least two parties: a `sender` and a `recipients`.

Here is an example of a `view` representing the sender's operations in the `transfer process`:  
This view is execute by the sender's FSC node.

```go
type TransferHouseView struct {
	*Transfer
}

func (d *TransferHouseView) Call(context view.Context) (interface{}, error) {
	// Prepare a new token transaction.
	tx, err := nftcc.NewAnonymousTransaction(
		context,
		nftcc.WithAuditor(
			view2.GetIdentityProvider(context).Identity("auditor"), // Retrieve the auditor's FSC node identity
		),
	)
	assert.NoError(err, "failed to create a new token transaction")

	buyer, err := nftcc.RequestRecipientIdentity(context, view2.GetIdentityProvider(context).Identity(d.Recipient))
	assert.NoError(err, "failed getting buyer identity")

	wallet := nftcc.MyWallet(context)
	assert.NotNil(wallet, "failed getting default wallet")

	// Transfer ownership of the house to the buyer
	house := &House{}
	assert.NoError(wallet.QueryByKey(house, "LinearID", d.HouseID), "failed loading house with id %s", d.HouseID)

	assert.NoError(tx.Transfer(wallet, house, buyer), "failed transferring house")

	// Collect signature from the parties
	_, err = context.RunView(nftcc.NewCollectEndorsementsView(tx))
	assert.NoError(err, "failed to collect endorsements")

	// Send to the ordering service and wait for confirmation
	_, err = context.RunView(nftcc.NewOrderingAndFinalityView(tx))
	assert.NoError(err, "failed to order and finalize")

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

## Queries

Here are two examples of view to list tokens.

The following view returns the list of unspent tokens:

```go
// GetHouse contains the input to query a house by id
type GetHouse struct {
	HouseID string
}

type GetHouseView struct {
	*GetHouse
}

func (p *GetHouseView) Call(context view.Context) (interface{}, error) {
	house := &House{}
	if err := nftcc.MyWallet(context).QueryByKey(house, "LinearID", p.HouseID); err != nil {
		if err == nftcc.ErrNoResults {
			return fmt.Sprintf("no house found with id [%s]", p.HouseID), nil
		}
		return nil, err
	}
	return house, nil
}
```

## Testing

To run the `Fungible Tokens` sample, one needs first to deploy the `Fabric Smart Client` and the `Fabric` networks.
Once these networks are deployed, one can invoke views on the smart client nodes to test the sample.

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
2. Alice and Bob have an Org2 Fabric Identity.

We can describe the network topology programmatically as follows:

```go
func Fabric(tokenSDKDriver string) []api.Topology {
	// Fabric
	fabricTopology := fabric.NewDefaultTopology()
	fabricTopology.EnableIdemix()
	fabricTopology.AddOrganizationsByName("Org1", "Org2")
	fabricTopology.SetNamespaceApproverOrgs("Org1")

	// FSC
	fscTopology := fsc.NewTopology()
	fscTopology.SetLogging("grpc=error:debug", "")

	issuer := fscTopology.AddNodeByName("issuer").AddOptions(
		fabric.WithOrganization("Org1"),
		fabric.WithAnonymousIdentity(),
		token.WithDefaultIssuerIdentity(),
	)
	issuer.RegisterViewFactory("issue", &views.IssueHouseViewFactory{})

	auditor := fscTopology.AddNodeByName("auditor").AddOptions(
		fabric.WithOrganization("Org1"),
		fabric.WithAnonymousIdentity(),
		token.WithAuditorIdentity(),
	)
	auditor.RegisterViewFactory("registerAuditor", &views.RegisterAuditorViewFactory{})

	alice := fscTopology.AddNodeByName("alice").AddOptions(
		fabric.WithOrganization("Org2"),
		fabric.WithAnonymousIdentity(),
		token.WithDefaultOwnerIdentity(tokenSDKDriver),
	)
	alice.RegisterResponder(&views.AcceptIssuedHouseView{}, &views.IssueHouseView{})
	alice.RegisterResponder(&views.AcceptTransferHouseView{}, &views.TransferHouseView{})
	alice.RegisterViewFactory("transfer", &views.TransferHouseViewFactory{})
	alice.RegisterViewFactory("queryHouse", &views.GetHouseViewFactory{})

	bob := fscTopology.AddNodeByName("bob").AddOptions(
		fabric.WithOrganization("Org2"),
		fabric.WithAnonymousIdentity(),
		token.WithDefaultOwnerIdentity(tokenSDKDriver),
	)
	bob.RegisterResponder(&views.AcceptIssuedHouseView{}, &views.IssueHouseView{})
	bob.RegisterResponder(&views.AcceptTransferHouseView{}, &views.TransferHouseView{})
	bob.RegisterViewFactory("transfer", &views.TransferHouseViewFactory{})
	bob.RegisterViewFactory("queryHouse", &views.GetHouseViewFactory{})

	tokenTopology := token.NewTopology()
	tokenTopology.SetSDK(fscTopology, &sdk.SDK{})
	tms := tokenTopology.AddTMS(fabricTopology, fabricTopology.Channels[0].Name, tokenSDKDriver)
	tms.SetTokenGenPublicParams("100", "2")
	fabric2.SetOrgs(tms, "Org1")
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

To help us bootstrap the networks and then invoke the business views, the `nft` command line tool is provided.
To build it, we need to run the following command from the folder `$GOPATH/src/github.com/hyperledger-labs/fabric-token-sdk/samples/fabric/nft`.

```shell
go build -o nft
```

If the compilation is successful, we can run the `nft` command line tool as follows:

``` 
./nft network start --path ./testdata
```

The above command will start the Fabric network and the FSC network,
and store all configuration files under the `./testdata` directory.
The CLI will also create the folder `./cmd` that contains a go main file for each FSC node.
The CLI compiles these go main files and then runs them.

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
./nft network clean --path ./testdata
```

The `./testdata` and `./cmd` folders will be deleted.

### Invoke the business views

If you reached this point, you can now invoke the business views on the FSC nodes.

To issue a nft token, we can run the following command:

```shell
./nft view -c ./testdata/fsc/nodes/issuer/client-config.yaml -f issue -i "{\"Address\":\"5th Avenue\", \"Valuation\":10, \"Recipient\":\"alice\"}"
```

The above command invoke the `issue` view on the issuer's FSC node. The `-c` option specifies the client configuration file.
The `-f` option specifies the view name. The `-i` option specifies the input data.
In the specific case, we are asking the issuer to issue a nft token for a house whose address is `5th Avenue` and its valuation is 10.
Owner of the token is `alice`.
If everything is successful, you will see something like the following:

```shell
"74f183f2-cbe1-4724-a6ea-4bbbc69bdc18"
```
The above is the NFT unique identifier.

Indeed, once the token is issued, the recipient can query its wallet to see the token.

```shell
./nft view -c ./testdata/fsc/nodes/alice/client-config.yaml -f queryHouse -i "{\"HouseID\":\"74f183f2-cbe1-4724-a6ea-4bbbc69bdc18\"}"
```

The above command will query Alice's wallet to get a list of unspent tokens whose type `TOK`.
You can expect to see an output like this (beautified): 
```shell
{
  "LinearID": "74f183f2-cbe1-4724-a6ea-4bbbc69bdc18",
  "Address": "5th Avenue",
  "Valuation": 10
}
```

Alice can now transfer some of her tokens to other parties. For example:

```shell 
./nft view -c ./testdata/fsc/nodes/alice/client-config.yaml -f transfer -i "{\"HouseID\":\"74f183f2-cbe1-4724-a6ea-4bbbc69bdc18\", \"Recipient\":\"bob\"}"
```

The above command instructs Alice's node to perform a transfer of 6 units of tokens `TOK` to `bob`.
If everything is successful, you will see something like the following:

```shell
"d1db2f7a7bd73e8dc4bb4b7c6785595157a5dbb60f00b9eeaed53f2c9e270c0f"
```

The above is the transaction id of the transaction that transferred the tokens.

Now, we check again Alice and Bob's wallets to see if they are up-to-date.

Alice:

```shell
./nft view -c ./testdata/fsc/nodes/alice/client-config.yaml -f queryHouse -i "{\"HouseID\":\"74f183f2-cbe1-4724-a6ea-4bbbc69bdc18\"}"
```

You can expect to see an output like this (beautified):

```shell
no house found with id [74f183f2-cbe1-4724-a6ea-4bbbc69bdc18]
```

Then, Bob:

```shell
./nft view -c ./testdata/fsc/nodes/bob/client-config.yaml -f queryHouse -i "{\"HouseID\":\"74f183f2-cbe1-4724-a6ea-4bbbc69bdc18\"}"
```

You can expect to see an output like this (beautified):

```shell
{
  "LinearID": "74f183f2-cbe1-4724-a6ea-4bbbc69bdc18",
  "Address": "5th Avenue",
  "Valuation": 10
}
```