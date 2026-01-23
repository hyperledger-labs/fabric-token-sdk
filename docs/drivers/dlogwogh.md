# DLOG w/o Graph Hiding

The `Zero Knowledge Asset Transfer DLog` (zkat-dlog, for short) driver supports privacy using Zero Knowledge Proofs.
We follow a simplified version of the blueprint described in the paper
[`Privacy-preserving auditable token payments in a permissioned blockchain system`](https://eprint.iacr.org/2019/1058)
by Elli Androulaki, Jan Camenisch, Angelo De Caro, Maria Dubovitskaya, Kaoutar Elkhiyaoui, and Björn Tackmann.
In more detail, the driver hides the token's owner, type, and quantity.
But it reveals which token has been spent by a given transaction. We say that this driver does not support `Token Identity Hiding` (previously known as `Graph Hiding`).
Owner anonymity is achieved by using Identity Mixer (Idemix, for short).
The identities of the issuers and the auditors are not hidden.

The above scheme is secure under `computational assumptions in bilinear groups` in the `random-oracle model`.

Let us now describe in more detail the implementation of the Driver API:

## Public Parameters and Manager

The relevant messages are [`here`](https://github.com/hyperledger-labs/fabric-token-sdk/tree/main/token/core/zkatdlog/nogh/protos).

## Issuer Service

The issuer service is responsible for the creation of an issue action.
Given in input:
- The public parameters,
- A token type,
- A list of tuples (value, owner) for the tokens to create,
  This service does the following:
- For each tuple for which a token must be created, it generates a Pedersen commitment containing the expected value and type.
- Given the openings of the commitments, it generates a ZK proof that proves that:
    - All the outputs have the same type, and
    - Each output value is in the expected value as defined by the public parameters.
- The action carries:
    - The output tokens;
    - The ZK proof;
    - The identity of the issuer that signs the token request;
- The metadata associated to the action contains:
    - The audit info of each involved identity;
    - The opening of each commitment;

## Transfer Service

The transfer service is responsible for the creation of a transfer action.
Given input:
- The public parameters,
- A list of tokens to spend named using their IDs,
- A list of tuples (value, owner) for the tokens to create,
  This service does the following:
- For each token to spend, it loads its commitment representation and its opening from the `TokensDB`.
  Recall that tokens appear in the `TokensDB` either because they were issued by an issuer or transferred from another owner.
  The fields that are relevant here are `ledger` and `ledger_metadata`.
- All the tokens that need to be spent must carry the same type.
- For each tuple for which a token must be created, it generates a Pedersen commitment containing the expected value and type.
- Then, it generates the ZK proof to prove that the commitments are valid Pedersen commitments under the given public params.
  Input and output tokens must have the same type, and the sum of values in input must be equal to the sum of the values in outputs.
  The value stored in each commitment must be in the allowed range.
- The action carries:
    - The input tokens spent;
    - The output tokens;
    - The ZK proof;
    - The identity of the issuer that signs the token request, in case it is needed (e.g., a redeem).
- The metadata associated to the action contains:
    - The audit info of each involved identity;
    - The opening of each commitment;

At this stage, no signature is generated to prove the will of the owners to spend the inputs.
Notice also that all the above crypto operations happen over the elliptic curve defined in the public parameters.

This happens later. The `ttx service` assists the developer to do so.

## Validator

The validator takes as input:
- The public parameters;
- A reference to the ledger to retrieve states, if needed.
- A serialized version of the Token Request to check against the public params and the ledger.

The `DLOG w/o Graph Hiding` validator is stateless, therefore it does not need access to the ledger.
The token request is marshalled using the `protobuf` protocol. The relative protobuf messages are [`here`](https://github.com/hyperledger-labs/fabric-token-sdk/tree/main/token/driver/protos/request.proto).
This guarantees backward and forward compatibility.

So, the validator unmarshals the serialized version of the token request.
The validator gets access to the serialized version of the actions.
The validator is equipped with an action deserializer to know how to deserialize actions.
Actions are also serialized using the `protobuf` protocol. The relative protobuf messages are [`here`](https://github.com/hyperledger-labs/fabric-token-sdk/tree/main/token/core/zkatdlog/nogh/protos).

Recall that we have two types of actions: Issue and Transfer.

For an issue action, the validator does the following:
- The action must be well-formed. All the expected fields must be there.
- It validates the ZK proof against the token commitments and the public parameters.
  The tokens must be valid Pedersen commitments under the public parameters.
  Moreover, all tokens must have the same type, and the value must be in the specific range.
- It checks that one of the issuers listed in the public parameters has signed the token request.
  Recall that an issuer identity is in the form of an X.509 certificate containing an ECDSA key.
  The message to verify is the ASN.1 encoding of the list of issue actions and the list of transfer actions concatenated with the anchor.
- Finally, it checks the metadata entries carried by the action. Only `public` metadata entries whose key has prefix `pub.` are allowed.

For a transfer action, the validator does the following:
- The action must be well-formed. All the expected fields must be there.
- The owner of each input must have signed the token request. Recall, this driver is not `Token Identity Hiding` (Graph Hiding), therefore the action carries the inputs being spent.
- If one of the outputs has an `empty` owner, this token signals a redeem operation. Therefore, at least one issuer in the list of the public params' issuers must have signed the token request.
- It validates the ZK proof against the token commitments and the public parameters.
  The input and output tokens must be valid Pedersen commitments under the public parameters.
  Moreover, all tokens must have the same type, and the value must be in the specific range.
  Finally, the sum of the input values must be equal to the sum of the output values.
- Finally, it checks the metadata entries carried by the action. Only `public` metadata entries whose key has prefix `pub.` are allowed.

Finally, the validator checks that at least one auditor in the list of public params' auditors has signed the token request.
This signature is carried in the `AuditorSignatures` field.

No secrets are involved during the validation process.
The only keys used are public keys of the issuers and the auditors that are listed in the public parameters.

## Key characteristics

- A token is represented on the ledger as the pair `(pedersen commitment to type and value, owner)`.
- Token metadata is a tuple containing: Token type, value, commitment blinding factor, and issuer's identity.
- The admissible values are in the range $[0..max-1]$, where $max$ is $2^{bitlength}$ and $bitlength$ is a public parameter. A typical value for $bitlength$ is $64$.
- The owner of a token can be:
    - An `Idemix Identity` to achieve identity anonymity. The public key of the Idemix Identity Issuer can be rotated.
    - An `HTLC-like Script` for interoperability;
    - A `Multisig Identity` for shared ownership;
- An issuer is identified by an X.509 certificate. The identity of the issuer is always revealed.
- Multiple issuers can be defined to issue a token type. Each such issuer can issue tokens of said type; This allows also for rotation of these keys.
- An auditor is identified by an X.509 certificate. The identity of the auditor is always revealed.
- Only one auditor is definable and its public key cannot be rotated.
- If an auditor is set, a request that doesn't carry its signature is considered invalid.
- Supported actions are: `Issue` and `Transfer`. `Redeem` is obtained as a `Transfer` that creates an output whose owner is `none`.
- An `Issue Action` proves that value is in the right range and one of the authorized issuers signed the request.
- A `Transfer Action` proves the following:
    - The sum of the inputs is equal to the sum of the outputs and the value of each output is in the valid range;
    - Inputs and outputs have the same type;
    - The owners of each input signed the request;
- The rightful owner of a token can redeem it;
- All the information required to operate the driver is found in the public parameters.
- Actions, public parameters, tokens, and token metadata are marshalled using `protobuf` messages.

In the coming sections, we give more details about the above key characteristics.

## Tokens and their Metadata

A token is presented on the ledger as a pair `(pedersen commitment to type and value, owner)`.
The Pedersen commitment is computed as $g_0^{H_{Z_r}(Type)}g_1^{Value} g_2^{BF}$,
where $H_{Zr}$ is the `hash to Zr` function provided by the bilinear group,
${g_i}$ are the bases of the Pedersen commitment,
$Type$ is a string, and both $Value$ and $BF$ are in $Z_r$.

The token metadata is then a tuple containing: Token type, value, blinding factor, and issuer's identity.

## Issue Action

An `issue action` is responsible for creating new tokens.
This is a privileged operation, meaning only an `issuer` can authorize it.

The action includes the following:
- The identity of the issuer;
- The output tokens to be created;
- A zero-knowledge proof of validity of the action;
- Additional application specific metadata that will be available on the ledger.

## Transfer Action

A `transfer action` spends existing tokens to create new tokens for an equivalent amount of value.
The ownership of the new tokens can be assigned to any admissible `owner identity`.

The action includes the following:
- The input tokens to be spent;
- The output tokens to be created;
- A zero-knowledge proof of validity of the action;
- Additional application specific metadata that will be available on the ledger.
- 

## Security

`DLOG w/o Graph Hiding` ensures token privacy, as well as anonymity and unlinkability of owner identities.
The validator guarantees the following:
*   **Issuer Authorization**: Only issuers listed in the public parameters can issue tokens.
*   **Auditor Authorization**: Only auditors listed in the public parameters can audit transactions.
*   **Owner Authorization**: Only legitimate owners can spend their tokens.
*   **Value Preservation**: In a transfer, the sum of inputs matches the sum of outputs.

**Limitations**:
*   It does not guarantee the anonymity of issuers and auditors.
*   It does **not** support Token Identity Hiding (Graph Hiding).

Detailed claims and security properties are discussed in the paper: [*Privacy-preserving auditable token payments in a permissioned blockchain system*](https://eprint.iacr.org/2019/1058).

### Secrets or Keys

Secrets and Keys in the Token SDK are associated with tokens and identities.
Depending on the driver implementation, the nature of these secrets and keys may vary.
Nevertheless, we can pinpoint where they are stored in the Token SDK stores.

Also, here, for simplicity, we assume that:
- The backend is Fabric,
- The Token Driver is `DLOG w/o Graph Hiding`.
- The Key Store is that provided by the Token SDK.

#### Tokens

A privacy-preserving token can be understood as a sealed envelope.
The content of the envelope/token is therefore a secret.
It is stored in the `Tokens` table (`ledger_metadata` field).
This secret is generated at the time of creation of a token when the Token Request is assembled
(see Section `Transfer Operation` for more details.).
This secret is stored in metadata section of the token request.
During the lifecycle of a token transaction, the token request is parsed, and all its components are stored in the DB.
Only the tokens whose token request in the `Requests` table has status `Valid` are actually stored in the `Tokens` table.
Indeed, only when the backend signals that a given token request is valid, its tokens are stored in the DB.

#### Identities

An identity can be understood as the public part of a cryptographic key-pair.
We have already seen that in the Token SDK there are identities used for different purposes.
Namely, to issue tokens, to own tokens, or to audit transactions.

To get to these identities, we need first a wallet.
Recall that a wallet is bound to a long-term cryptographic key-pair.
The `IdentityConfigurations` table contains the information about the identities that can be used to derive wallets.

So, let us ask: How does this table get populated?
There are two ways:
- From the configuration file. Here is an example taken from [`here`](../core-token.md):
```yaml
      # sections dedicated to the definition of the wallets
      wallets:
        # Default cache size reference that can be used by any wallet that support caching
        defaultCacheSize: 3
        # owner wallets
        owners:
        - id: alice # the unique identifier of this wallet. Here is an example of use: `ttx.GetWallet(context, "alice")`
          default: true # is this the default owner wallet
          # path to the folder containing the cryptographic material associated to wallet.
          # The content of the folder is driver dependent
          path:  /path/to/alice-wallet
          # Cache size, in case the wallet supports caching (e.g. idemix-based wallet)
          cacheSize: 3
        - id: alice.id1
          path: /path/to/alice.id1-wallet
        # issuer wallets
        issuers:
          - id: issuer # the unique identifier of this wallet. Here is an example of use: `ttx.GetIssuerWallet(context, "issuer")`
            default: true # is this the default issuer wallet
            # path to the folder containing the cryptographic material associated to wallet.
            # The content of the folder is driver dependent
            path: /path/to/issuer-wallet
            # additional options that can be used to instantiated the wallet.
            # options are driver dependent. With `fabtoken` and `dlog` drivers,
            # the following options apply.
            opts:
              BCCSP:
                Default: SW
                SW:
                  Hash: SHA2
                  Security: 256
                # The following only needs to be defined if the BCCSP Default is set to PKCS11.
                # NOTE: in order to use pkcs11, you have to build the application with "go build -tags pkcs11"
                PKCS11:
                  Hash: SHA2
                  Label: null
                  Library: null
                  Pin: null
                  Security: 256
        # auditor wallets
        auditors:
          - id: auditor # the unique identifier of this wallet. Here is an example of use: `ttx.GetAuditorWallet(context, "auditor")`
            default: true # is this the default auditor wallet
            # path to the folder containing the cryptographic material associated to wallet.
            # The content of the folder is driver dependent
            path: /path/to/auditor-wallet
            # additional options that can be used to instantiated the wallet.
            # options are driver dependent. With `fabtoken` and `dlog` drivers,
            # the following options apply
            opts:
              BCCSP:
                Default: SW
                PKCS11:
                  Hash: SHA2
                  Label: null
                  Library: null
                  Pin: null
                  Security: 256
                SW:
                  Hash: SHA2
                  Security: 256
```
For each wallet type, one defines a list of identities that will be loaded into the `IdentityConfigurations` table.
The secret keys are assumed to be on a file system or inside an HSM, when supported.
Currently, we only supports ECDSA keys in HSM.
At loading time, the configuration is parsed and the content of filesystem is loaded inside the `IdentityConfigurations` table.
The corresponding secret keys are either stored in the `KeyStore` table or in an external custom defined key store.
After the first loading, the content of the file system can be removed.
- Directly by loading the `IdentityConfigurations` table with the relevant information.

We currently supports two types of keys: ECDSA and Idemix.
When loading an ECDSA key inside the `IdentityConfigurations`, only the corresponding `X.509` certificate is loaded inside the table.
The secret key is stored in the key store.
When loading Idemix credentials, only the public part of the credential is store in the `IdentityConfigurations` table.
The secret key is stored in the key store and the secret key component of the credential is replaced with the secret key's SKI.

Idemix identities are only used for the `owner` wallets. X.509 identities are used for issuers and auditors.
When an identity is derived from an owner wallet via a call to `GetRecipientIdentity`,
then both the `IdentityInfo` and the `IdentitySigners` tables are filled with a row.
The field `identity_audit_info` contains private information about the identity, such as the `enrollment id`, the `rovocation id`, used to revoke that identity, and so on.
For an X.509 identity, the identity itself already reveals everything.
For an Idemix identity (or pseudonym), it contains the secrets to de-anonymise the pseudonym.
The `info` field in the `IdentitySigners` table remains empty as well as `token_metadata` and `token_metadata_audit_info`.
The `KeyStore` table contains a row whose `key` field gets the value of the SKI of the cryptographic key stored and value is the cryptographic key itself.

#### Identity and Access Management

The Token SDK uses identities to establish trust and control access within the system.
These identities are like digital passports that verify who a party is and what actions they're authorized to perform.
Therefore, the Token SDK's identity service offers:
1. A way to define these identities,
2. A way to generate and verify digital signatures valid under these identities, and
3. A way to marshal/unmarshal identities and signatures.

#### Roles as Teams

Within the identity service, long-term identities are grouped into roles.
Imagine these roles as teams with specific permissions.
Here are some key roles:

* **Issuers:** Like a mint that creates coins, issuers have the power to create new tokens.
* **Owners:** Owners hold tokens, just like possessing a digital asset.
* **Auditors:** These act as financial inspectors, overseeing token requests and ensuring proper use.
* **Certifiers:** They verify the existence and legitimacy of specific tokens, similar to checking identification.
  This role is used only by certain token drivers that support the so-called `Token Identity Hiding` (Graph Hiding).

Recall that long-term identities are stored in the `IdentityConfigurations`.
The corresponding secrets are stored either in the `KeyStore` table or an external key store, if configured.

#### Identity Options: Passports and Beyond

Identities come in different forms.

A common choice is an `X.509 certificate`, a secure electronic passport that links a real-world entity (website, organization, or person) to a public key using a digital signature.
This certificate uniquely identifies the entity it represents.

Another option is `anonymous credentials`, which allow users to prove they have certain attributes (like age or qualifications) without revealing their entire identity.
Imagine showing an ID that only displays relevant information, not your full details.
This is particularly useful for protecting user privacy.

#### Roles and Wallets: Managing Access

Importantly, roles can contain different identity types.
These roles then act as the foundation for creating wallets.
A wallet is like a digital vault that stores a long-term identity (the main key) and any credentials derived from it, if supported.
Different wallet types provide different functionalities.
For example, an issuer wallet lets you see a list of issued tokens, while an owner wallet shows unspent tokens.
All these wallets have though a minimum common denominator:
Given a wallet, you, as the developer, can derive identities to be used in different contexts.
For example, given an owner wallet, you can derive the identities that will own tokens.
To spend these tokens, the wallet will give access to the signers bound to these identities.
The signers can be used to generate the signatures necessary to spend these tokens.
