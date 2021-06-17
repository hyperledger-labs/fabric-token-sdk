# The Fabric Token SDK

The scope of the `Fabric Token SDK` is to deliver a set of API and services that let developers create token-based 
distributed application on Hyperledger Fabric.
The `Fabric Token SDK` has the following characteristics;
- It adopts the UTXO model. In the UTXO model, a direct acyclic graph reflects the movements of the assets. 
  Nodes are token transactions. Edges are transaction outputs. Each new token transaction consumes some the 
  UTXOs and create new ones.
- Wallets contain a set of `secret keys` and keep track of the list of unspent outputs `owned` those keys.
- It supports different privacy levels: from a `plain` instantiation, where everything is in the clear on the ledger, 
  to `Zero Knowledge-based` instantiations that will obfuscate the ledger while enforcing the required invariants.
- It allows the developers to write their own `services` on top of the Token SDK to deliver customised compoenents 
  for their token-based applications.

## The Token SDK Stack

This is the Fabric Token SDK stack: 

![stack](imgs/stack.png)

It consists of the following layers:
- `Services` (light-blue boxes): Services offer pre-packaged token-related functionalities,
like `Token Transaction` assembling, `Token Selectors`, and so on.
They are built of top of the `Token API` abstraction. Therefore, they are independent of the underlying token technology.
- `Token API`: This API offers a useful abstraction to deal with tokens in an implementation and blockchain independent way. 
- `Driver API`: This API takes the burden of translating calls to the Token API into API calls that are implementation-specific.
- `Driver Implementations`: This is the lowest level of the Token SDK. A driver implementation is responsible for 
  defining the representation of tokens on the ledger, what it means to perform certain token actions,
  and when a token transaction is valid, among other things.
  
The `Fabric Token SDK` is built on top of the `Fabric Smart Client` stack. 
The `Smart Client` allows the `Token SDK` to: 
- Orchestrate very complex token-dependent business processes;
- Store the tokens inside the Vault for easy lookup and manipulation;
- To listen to events from Fabric related to token transaction, and more.

Le us explore in more details each layer of the Token SDK stack.
- [`Token API`](./token-api.md): The `Token API` offers a useful abstraction to deal with tokens in an
  implementation and blockchain independent way. 
- [`Driver API`](./driver-api.md): The Driver API defines the contracts any implementation should respect to 
  be compatible with the Token API.
- [`Driver Implementations`](./drivers.md): The Token SDK comes equipped with two driver implementations:
  - [`FabToken`](./fabtoken.md): This is a simple implementation of the Driver API that does not support privacy. 
  - [`ZKAT DLog`](./zkat-dlog.md): This driver supports privacy via Zero Knowledge. We follow
    a simplified version of the blueprint described in the paper
    [`Privacy-preserving auditable token payments in a permissioned blockchain system`]('https://eprint.iacr.org/2019/1058.pdf')
    by Androulaki et al.
- [`Services`](./services.md): It is at `service layer` that we will describe the integration with Fabric. 
  In particular, we will focus our attention on the lifecycle of a `Fabric Token Transaction`. 
  This will give us the chance to touch many building blocks.