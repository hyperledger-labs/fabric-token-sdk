# Token API

The `Token API`, located in `token` and `token\token` packages, 
offers a useful abstraction to deal with tokens in an implementation and backend independent way.

The Token-SDK handles tokens that consist of the following triplet:
- `Owner`: The owner of the token; Each driver implementation can interpreter this field as needed. It can be a public-key, a script,
  anything the underlying specific driver supports.
- `Type`: The *denomination* of the token;
  This is a string whose value can be application specific. Examples are:
  The denomination of a digital currency or unique identifiers.
- `Quantity`: The amount stored by this token. It is a positive number, larger or equal to zero, 
  encoded as a string containing a number in base 16. The string starts with the prefix `0x`.

These tokens are `fungible` with the respect to the same type. 
In particular, tokens with the same denomination can be merged and split, if not otherwise forbidden.

The above definition allows the developers to define non-fungible tokens as well.
A non-fungible token is a token whose quantity is `1` and whose type is `unique`. 
If uniqueness is guaranteed, then such a token is by all means a non-fungible token.
Drivers are free to implement additional semantics for non-fungible tokens.

A token can be spent only by the `rightful owner`. This concept is implementation dependant. 
For example,
if the `Owner` field contains a public-key, then a valid signature under that public key must be presented to spend the token.
If the `Owner` field contains a script, then an input that satisfies the script must be presented to spend the token.

The Token SDK supports the following basic operations:
- The `Issue` operation creates new tokens. `Issuers` are in charge of issuing new tokens. Depending on the driver
  implementation, an issuing policy can be used to identify the authorized issuers for a given type. 
- The `Transfer` operation transfers the ownership of a given token. A transfer operation must refer to tokens of the same
type.
- The `Redeem` operation deletes tokens. Depending on the driver implementation, either the rightful owner or special
parties, called `redeemers`, can invoke this operation.

A `Token Request` aggregates token operations that must be performed atomically.

Let us now focus on the some of the main building blocks the `Token API` consists of:

![token_api.png](imgs/token_api.png)

## Token Management Service

The `Token Management Service` (`token.ManagementService`) (TMS, for short) is the entry point of the Token SDK
and gives access to the rest of the APIs.
The tuple `(network, channel, namespace, public parameters)` uniquely identifies a TMS, where:
- `network` is the identifier of the network/backend of reference;
- `channel` is the channel inside the network. If not available, it is empty;
- `namespace` is the namespace, inside the channel, that stores the tokens.
- `public parameters` contain all information needed to operate the specific token infrastructure.

The developer can get an instance of the `Token Management Service` by using the `GetManagementService` function:
```go
    tms := token.GetManagementService(context)
```
where `context` is an `FSC View Context`.
The developer can pass additional options to request a specific TMS like:
```go
    tms := token.GetManagementService(context, token.WithNetwork("my-network"))
```

## Public Parameters Manager

Each TMS is associated to some public parameters that contain all information needed to operate the token infrastructure.
The `Public Parameters Manager` (`token.PublicParametersManager`) is responsible for handling the public parameters.
Even though, parts of the public parameters are driver-specific, we can identify the following common information:
    - `Precision`: The precision used to represent the token quantity.
    - `MaxTokenValue`: It is the maximum quantity that a token can contain.
    - `TokenDataHiding`: When true it means that the content of the tokens is hidden.
    - `GraphHiding`:  When true it means that the tokens are untraceable.
    - `Auditors`: A list of auditor identities.

The developer can access the `Public Parameters Manager` of a give TMS as follows:
```go
    ppm := tms.PublicParametersManager()
```

## Wallet Manager

A Wallet consists of a long-term identity and all its derivation (if any).
Examples of long-term identities are:
    - An `X509 Certificate` for an ECDSA signing public-key
    - An `Idemix Credential`. In this case, the wallet will contain also all pseudonyms derived from the credential.
  
However, it is always the specific driver that dictates what a long-term identity is.

All operations that require a signature refer to wallets to identify the signing and verification keys.
There are wallets for `Issuers`, `Owners`, and `Auditors`.
Depending on the nature of the wallet additional information can be extracted like:
    - An Issuer Wallet gives access to the list of issued tokens;
    - An Owner Wallet gives access to the list of owned unspent tokens;

The Wallet Manager (`token.WalletManager`) helps managing these wallets.

The developer can access the Wallet Manager of a given TMS as follows:
```go
    wm := tms.WalletManager()
```

## Token Request

The Token Request (`token.Request`) is a container of token actions (issue, transfer, and redeem) that must be
performed atomically. The Token Request offers API to add actions or inspect actions already present in the container.

The developer can create a new Token Request as follows:
```go
    tr, err := token.NewRequest()
```
or unmarshal a previously marshalled Token Request as follows:
```go
    tr, err := token.NewRequestFromBytes(achor, actions, metadata)
```

Looking ahead, parties interacting to assemble a token transaction are, under the hood, assembling a `Token Request` that it is
later marshalled into the format required by the target backend.

This is the anatomy of a Token Request:

![token_request.png](imgs/token_request.png)

It consists of three parts:
- `Anchor`: It is used to bind the Actions to a given Transaction. In Fabric, the anchor is the Transaction ID.
- `Actions`: It is a collection of `Token Action`:
    - `Issues`, to create new Tokens;
    - `Transfers`, to manipulate Tokens (e.g., transfer ownership or redeem)

  The actions in the collection are independent. An action cannot spend tokens created by another action in the same Token Request.
  In addition, actions comes with a set of `Witnesses` to verify the `right to spend` or the `right to issue` a given token.
  In the simplest case, the witnesses are the signatures of the issuers and the token owners.

- `Metadata`: It is a collection of `Token Metadata`, one entry for each Token Action.
  Parties, assembling a token request, exchange metadata that contain secret information used by
  the parties to check the content of the token actions. This is particularly relevant when using ZK-based drivers.
  Notice that, the ledger does not store any metadata.

Looking ahead: As we mentioned earlier, a Token Request is itself agnostic to the details of the specific backend.
Indeed, a Token Request must be translated to the Transaction format of the target backend to become meaningful.
A service called `Token Request Translator` translates the token requests.
The `Token Request Translator` does not belong to the Token API. It is offered as a service on top of the `Token API`
because it is backend dependant. 

Here is a pictorial representation of the translation process for Fabric

![token_request_translator.png](imgs/token_request_translator.png)

In Fabric, it is the `Token Chaincode` that performs validation and translations of token requests.
More information [`here`](./services.md).

## Validator 

The validator (`token.Validator`) is the component that sets the validation rules for a Token Request. The rules depend on the
type of tokens supported (fungible, non-fungible, and so on), and on the specific driver implementation.
A Validator validates Token Requests with the respect to:
- A given Anchor (e.g., Fabric TxID), and
- The Ledger. Notice that, in certain implementations, the ledger might not be needed.

Even though, certain validation rules are driver specific, we can identify the following general validation rules.
A token request should:
- Well-formed, and
- Satisfies the constraints of the payment system. Namely:
- Only the `rightful owner` can transfer a token,
- No token can be created out of the blue,
- Audited(able)
- Etc. (Each implementation can enforce additional requirements, if needed)

## Vault

The vault (`token.Vault`) gives access to the tokens that are owned by the wallets in the wallet manager.

The developer can access the vault of a given TMS as follows:
```go
    vault := tms.Vault()
```

## Selector Manager

A token selector (`token.Selector`) allows the developer to select tokens from the vault under certain conditions.
For example, a token selector can be used to select tokens of a given type and for a given amount that are owned by a specific wallet,
The selector must ensure that two different routines receives different tokens. This is important to avoid double-spending.
This means that selected tokens are locked until the transaction that uses them is committed or rejected, or a timeout occurs,
or an explicit unlock operation is performed.

The Selector Manager (`token.SelectorManager`) returns selector instances.

The developer can access the Selector Manager of a given TMS as follows:
```go
    sm := tms.SelectorManager()
``` 

and get a new selector, for a given id, as follows:

```go
    selector := sm.NewSelector(tr.Anchor)
```

Notice that, the tokens selected by this selector will be locked under the passed id.

## Signature Service

The Signature Service (`token.SignatureService`) is the component that gives access to signature verifiers and signers
bound to identities obtained from the Wallet Manager.

The developer can access the Signature Service of a given TMS as follows:
```go
    sigService := tms.SignatureService()
```

## Config Manager

The Config Manager (`token.ConfigManager`) is the component that gives access to the configuration of the TMS.

The developer can access the Config Manager of a given TMS as follows:
```go
    cm := tms.ConfigManager()
```