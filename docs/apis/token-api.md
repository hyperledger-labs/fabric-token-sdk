## Unleashing the Power of Tokens: A Look at the Token API

The Token API, housed within the `token` and `token/token` packages, provides a powerful and versatile way to manage tokens across different implementations and backends. It acts as an abstraction layer, simplifying token interaction for developers.

This API handles tokens defined by a three-part structure:

* **Owner:** This identifies the token's rightful owner. Driver implementations can interpret this field based on their specific needs. It could represent a public key, a script, or anything the underlying driver supports.
* **Type:** Think of this as the token's denomination, a string value specific to your application. Examples include digital currency denominations or unique identifiers.
* **Quantity:** This represents the amount stored by the token. It's always a non-negative number encoded as a string in base 16, prefixed with "0x".

Tokens of the same type are considered **fungible**. This means they can be merged or split (unless restricted), similar to how interchangeable units of currency behave. However, the API also allows for the creation of **non-fungible tokens**. These unique tokens have a quantity of 1 and a unique type. Drivers can further enhance non-fungible token functionality with additional features.

Spending a token requires authorization from the rightful owner. This process depends on the implementation. For instance, if the Owner field holds a public key, a valid signature using that key is necessary. Script-based tokens require an input that satisfies the script's conditions.

The Token API empowers developers with essential operations:

* **Issue:** Creates new tokens. The designated issuers, determined by driver-specific issuing policies, control this operation.
* **Transfer:** Shifts ownership of a token. Transfers can only occur between tokens of the same type.
* **Redeem:** Deletes tokens. Depending on the driver, either the owner or designated redeemers can perform this action.

**Token Requests** bundle these operations, ensuring they are executed atomically, meaning all operations succeed or fail together.

Now, let's delve deeper into the core components that make up the Token API...

![token_api.png](../imgs/token_api.png)

## Diving into the Token Management Service (TMS)

The Token Management Service (TMS), also known as `token.ManagementService`, acts as the central hub for the Token SDK. 
It provides access to all the other APIs within the SDK.

Imagine a TMS as being uniquely identified by a four-part address:

1. **Network:** This identifies the underlying network or backend system.
2. **Channel (Optional):** This specifies the channel within the network. If not applicable, it remains empty.
3. **Namespace:** This defines the specific namespace within the channel where tokens are stored.
4. **Public Parameters:** This section holds all the crucial information needed to operate the particular token infrastructure.

To interact with the TMS, developers can leverage the `GetManagementService` function:

```go
tms := token.GetManagementService(context)
```

Here, `context` refers to an `FSC View Context` ([https://github.com/hyperledger-labs/fabric-smart-client](https://github.com/hyperledger-labs/fabric-smart-client)), which provides essential environmental details.

For more granular control, developers can provide additional options with the function. For example:

```go
tms := token.GetManagementService(context, token.WithNetwork("my-network"))
```

This allows you to specify a particular network named "my-network" for the TMS instance.

## Unveiling the Public Parameters Manager

Every Token Management Service (TMS) is linked to a set of public parameters. 
This information, managed by the `Public Parameters Manager` (`token.PublicParametersManager`), holds the key to operating the token infrastructure effectively.

While some parameters are specific to different drivers, some common details are included:

* **Precision:** This dictates the level of detail used to represent the amount stored in a token.
* **MaxTokenValue:** This sets a limit on the maximum quantity a single token can hold.
* **Token Data Hiding:** When enabled (true), the content of the tokens is obscured.
* **Graph Hiding:** With this set to true, tokens become untraceable within the system.
* **Auditors:** This list identifies authorized auditors for the token system.

To access the `Public Parameters Manager` associated with a specific TMS, developers can use the following code:

```go
ppm := tms.PublicParametersManager()
```

This line retrieves the manager instance from the provided TMS object.

## A Look Inside Wallets

A Wallet acts like a digital identity vault, holding a long-term identity (think of it as a main key) and any credentials derived from it. 
This identity can take different forms, such as an X509 Certificate for signing purposes or an [`Idemix Credential`](https://github.com/IBM/idemix) with its associated pseudonyms. 
Ultimately, the specific driver you're using determines what constitutes a valid long-term identity.

Wallets play a crucial role in signing and verifying operations. 
Whenever a signature is needed, the system looks to the appropriate wallet within the Wallet Manager to locate the necessary keys. 
This manager keeps track of wallets for different roles like Issuers, Owners, and Auditors. 
Notably, Certifiers aren't supported because this driver doesn't handle a specific privacy feature called Graph Hiding.

Depending on the type of wallet, you can extract additional information. 
For instance, an Issuer Wallet lets you see a list of issued tokens, while an Owner Wallet shows you their unspent tokens.

The Wallet Manager, conveniently referred to as `token.WalletManager` in code, serves as the central hub for managing all these wallets.

Here's how developers can access the Wallet Manager of a specific TMS (Token Management System):

```go
wm := tms.WalletManager()
```

## Building a Token Transaction

Imagine a `Token Request` as a blueprint for a secure financial transaction. 
It groups together different actions like issuing new tokens, transferring ownership, or redeeming existing ones. 
These actions must happen all at once, ensuring everything goes smoothly.

The `Token Request` offers a toolbox for developers to easily add or review the actions included in this blueprint.

Here's a breakdown of its key components:

* **Anchor:** This acts like a reference point, tying the actions to a specific transaction within the system. In Hyperledger Fabric, the anchor corresponds to the transaction ID.
* **Actions:** This is the heart of the blueprint, containing a set of specific token operations:
    * **Issue:** Creates brand new tokens.
    * **Transfer:** Manages existing tokens, allowing ownership changes or redemption.

These actions within the request are independent. One action cannot utilize tokens created by another action within the same request. Additionally, each action comes with witnesses, which are essentially verifications. These witnesses confirm the "right to spend" or "right to issue" a particular token. In simpler scenarios, witnesses might be signatures from token owners or issuers.
* **Metadata:** This serves as a secure communication channel between involved parties. It holds secret information that allows them to verify the details of the token actions. This is especially important when using privacy-focused drivers based on Zero-Knowledge proofs. Importantly, the ledger itself doesn't store this metadata.

Developers can create a new `Token Request` from scratch using this line of code:

```go
tr, err := token.NewRequest()
```

Alternatively, they can bring a previously saved request back to life using this approach:

```go
tr, err := token.NewRequestFromBytes(achor, actions, metadata)
```

Behind the scenes, when parties collaborate to create a token transaction, they're essentially building a `Token Request`. This request is then translated into a format that the specific ledger system (like Hyperledger Fabric) understands. Remember, a `Token Request` itself is independent of the underlying ledger. To be processed, it needs a translation service called the `Token Request Translator`. This translator converts the request into the transaction format specific to the chosen ledger backend. Since this translation depends on the ledger being used, the `Token Request Translator` is a separate service on top of the core `Token API`.

Here's a visual representation of the translation process for Hyperledger Fabric (image not included, but the concept remains).

In Fabric, a special component called the `Token Chaincode` is responsible for validating and translating these token requests.

The Token SDK provides a handy service, the `ttx service`, to streamline working with token requests as transactions. 
This service takes care of the entire process, from creation to completion. 
For a deeper dive into this service, check out the dedicated page: [`ttx service`](../services/ttx.md).

## Validator 

The `token.Validator` acts as the guardian of token requests, ensuring they adhere to specific rules. 
These rules vary depending on the types of tokens supported (fungible or non-fungible) and the chosen driver implementation.

The validator meticulously examines each token request against two key aspects:

- The provided anchor (think of it as a reference point, like a transaction ID in Fabric).
- The target ledger (though some implementations might not require the ledger itself).

While specific validation rules can differ based on the driver, some general principles hold true. A valid token request should:

- Be structurally sound (well-formed).
- Align with the constraints of the payment system. This means:
    - Only authorized owners can transfer tokens.
    - Tokens can't be conjured out of thin air (they must be issued properly).
    - The system should be auditable (transactions can be traced and verified).
- Additional requirements can be enforced by individual implementations as needed.

## Manage all your tokens with ease using Token Vault

Token Vault is your central hub for everything token-related. 
It works seamlessly across different systems, offering a comprehensive toolkit to understand your token holdings.

With Token Vault, you can:

* **Gain instant insights:** See all your tokens in one place, check their status, and get detailed information about each one.
* **Track transactions:** Easily query if a transaction is pending or confirm ownership of a specific token.
* **Explore unspent tokens:** Utilize iterators to discover tokens you haven't used yet.
* **List all tokens:** Get a complete overview of all your tokens or those issued on the network.
* **Dive deeper:** Retrieve specific details about tokens and their outputs. (For certain systems) You can even find out who deleted a token, if applicable.

**Easy access for developers:** Developers can effortlessly access the vault of a specific Token Management System (TMS) using a simple line of Go code:

```go
vault := tms.Vault()
```

This vault instance is the same one provided by the dedicated [`Vault Service`](../services/vault.md).

## Token Selector Manager

The token SDK empowers developers to select specific tokens from the vault for transactions using the `token.Selector` function. 
This selector acts like a refined filter, allowing you to specify conditions like token type, amount, and even ownership. 
To safeguard against double-spending, any chosen token is automatically locked until the transaction's fate is sealed – be it commitment, rejection, timeout, or manual unlock. 
This ensures developers leverage the appropriate tokens while eliminating double-spending woes.

Let's delve deeper. 
To get a selector for your specific needs, simply call the `SelectorManager()` method on your TMS instance. 
This manager provides access to new selectors using the `NewSelector(tr.Anchor)` function.
Remember, the chosen tokens will be locked under the provided ID.

Behind the scenes, these selectors are meticulously crafted by the dedicated [`Token Selector Service`](../services/selector.md).

## Signature Service

The `token.SignatureService` acts as your gateway to secure transactions. 
It provides access to both signature verifiers and signers, all seamlessly linked to identities retrieved from the Wallet Manager.

Getting started is a breeze. 
Simply call the `SigService()` method on your existing TMS instance, and you'll be ready to leverage these powerful signing and verification tools.

## Config Manager

Unveil the inner workings of your TMS with the `token.ConfigManager`. 
This component grants you access to the TMS's configuration, giving you full control over its behavior.

To interact with the configuration manager for a specific TMS instance, simply use the `tms.ConfigManager()` method. It's that easy!
