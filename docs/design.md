# The Fabric Token SDK

The scope of the `Fabric Token SDK` is to deliver a set of `APIs` and `Services` that let developers create token-based 
applications on Hyperledger Fabric, Orion, and potentially more.
The `Fabric Token SDK` has the following characteristics;
- It adopts the `UTXO model`. In the UTXO model, a direct acyclic graph reflects the movements of the assets. 
  Nodes are token transactions. Edges are transaction outputs. Each new token transaction consumes some the 
  UTXOs and create new ones.
- Key-Management via `Wallets`. A Wallet contains a set of `secret keys` and keeps track of the list of unspent outputs `owned` by those keys.
- It supports `multplie privacy levels`: from a `plain` instantiation, where everything is in the clear on the ledger, 
  to `Zero Knowledge-based` instantiations that will obfuscate the content of the token transactions on the ledger while enforcing the required invariants
  (see [drivers](./drivers.md) for more information).
- It allows developers to write their own `Services` on top of the Token SDK API to deliver customised components 
  for token-based applications.

## The Token SDK Stack

This is the Fabric Token SDK stack: 

![stack](imgs/stack.png)

It consists of the following layers (from the top):
- [`Services`](./services.md): Services offer pre-packaged token-related functionalities,
  like `Token Transaction` assembling, `Token Selectors` of unspent tokens, and so on.
  They are built of top of the `Token API` abstraction. Therefore, they are independent of the underlying token technology.
- [`Token API`](./token-api.md): This API offers a useful abstraction to deal with tokens in an implementation and blockchain independent way.
  Tokens and the related operations are represented in a meta-language that it is then translated to a given backend (Fabric, Orion, etc...).  
- [`Driver API`](./driver-api.md): This API takes the burden of translating calls to the Token API into API calls that are token implementation-specific.
  Indeed, a transfer operation with privacy via Zero Knowledge is not the same as a transfer operation with privacy via plain instantiation.
- [`Drivers`](./drivers.md): This is the lowest level of the stack. A driver is responsible for 
  defining the representation of tokens, what it means to perform certain token operations,
  and when a token transaction is valid, among other things.
  
The `Fabric Token SDK` is built on top of the `Fabric Smart Client` stack. 
The `Smart Client` allows the `Token SDK` to: 
- Orchestrate very complex token-dependent business processes via [`views`](https://github.com/hyperledger-labs/fabric-smart-client/blob/main/docs/view/api.md);
- Store the tokens inside the Vault for easy lookup and manipulation;
- To listen to events from the backends related to token transaction, and more.
