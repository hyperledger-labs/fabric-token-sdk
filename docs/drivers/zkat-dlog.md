# Zero Knowledge Asset Transfer DLog Driver

The `Zero Knowledge Asset Transfer DLog` (zkat-dlog, for short) driver supports privacy using Zero Knowledge Proofs. 
We follow a simplified version of the blueprint described in the paper <!-- markdown-link-check-disable -->
[`Privacy-preserving auditable token payments in a permissioned blockchain system`]('https://eprint.iacr.org/2019/1058.pdf')<!-- markdown-link-check-disable -->
by Elli Androulaki, Jan Camenisch, Angelo De Caro, Maria Dubovitskaya, Kaoutar Elkhiyaoui, and Björn Tackmann.
In more details, the driver hides the token's owner, type, and quantity.
But it reveals which token has been spent by a given transaction. We say that this driver does not support `graph hiding`.
Owner anonymity is achieved by using Identity Mixer (Idemix, for short).
The identities of the issuers and the auditors are not hidden.

The above scheme is secure under `computational assumptions in bilinear groups` in the `random-oracle model`.

The driver implementation is available under the folder [`nogh/v1`](./../../token/core/zkatdlog/nogh/v1).

## Key characteristics

- A token is represented on the ledger as the pair `(pedersen commitment to type and value, owner)`.
- A token metadata is a tuple containing: Token type, value, commitment blinding factor, and issuer's identity.
- The admissible values are in the range $[0..max-1]$, where $max$ is $2^{bitlength}$ and $bitlength$ is a public parameter. A typical value for $bitlength$ is $64$.
- The owner of a token can be:
  - An `Idemix Identity` to achieve identity anonymity. The public key of the Idemix Identity Issuer can be rotated.
  - An `HTLC-like Script` for interoperability;
  - A `Multisig Identity` for shared ownership;
- An issuer is identified by an X509 certificate. The identity of the issuer is always revealed.
- Multiple issuers can be defined to issue a token type. Each such an issuer can issue tokens of said type; This allows also for rotation of these keys.
- An auditor is identified by an X509 certificate. The identity of the auditor is always revealed.
- Only one auditor is definable and it is public key cannot be rotated.
- If an auditor is set, a request that doesn't carry its signature is considered invalid.
- Supported actions are: `Issue` and `Transfer`. `Reedem` is obtained as a `Transfer` that creates an output whose's owner is `none`.
- An `Issue Action` proves that value is in the right range and one of the authorized issuers signed the request. 
- A `Transfer Action` proves the following:
  - The sum of the inputs is equal to the sum of the outputs and the value of each output is in the valid range;
  - Inputs and outputs have the same type;
  - The owners of each input signed the request;
- The rightful owner of a token can redeem it;
- All the information required to operate the driver are found in the public parameters.
- Actions, public parameters, tokens, and tokens metadata are marshalled using `protobuf` messages.

In the coming sections, we give more details about the above key characteristics.

## Tokens and their Metadata

A token is presented on the ledger as a pair `(pedersen commitment to type and value, owner)`.
The pedersen commitment is computed as $g_0^{H_{Z_r}(Type)}g_1^{Value} g_2^{BF}$, 
where $H_{Zr}$ is the `hash to Zr` function provided by the bilinear group,
${g_i}$ are the bases of the pedersen commitment,   
$Type$ is a string, and both $Value$ and $BF$ are in $Z_r$.

The token metadata is then a tuple containing: Token type, value, blinding factor, and issuer's identity.  

The code that models the above concepts can be found under [`v1/token`](./../../token/core/zkatdlog/nogh/v1/token).
The protobuf messages for `Token` and `TokenMetadata` can be found under ['nogh/protos'](./../../token/core/zkatdlog/nogh/protos/noghactions.proto).

## Issue Action

An `issue action` is responsible for creating new tokens.  
This is a privileged operation, meaning only an `issuer` can authorize it.

The action includes the following:
- The identity of the issuer;
- The output tokens to be created;
- A zero-knowledge proof of validity of the action;
- Additional public metadata.

The code that contains the definition of the action and the prover/verifier code can be found under [`v1/issue`](./../../token/core/zkatdlog/nogh/v1/issue).
The code that assembles the issue action can be found here: [`issue.go`](./../../token/core/zkatdlog/nogh/v1/issue.go).
The protobuf messages for the action can be found under ['nogh/protos'](./../../token/core/zkatdlog/nogh/protos/noghactions.proto).

## Transfer Action

A `transfer action` spends existing tokens to create new tokens for an equivalent amount of value. 
The ownership of the new tokens can be assigned to any admissible `owner identity`.

The actions includes the following:
- The input tokens to be spent;
- The output tokens to be created;
- A zero-knowledge proof of validity of the action;
- Additional public metadata.

The code that contains the definition of the action and the prover/verifier code can be found under [`v1/transfer`](./../../token/core/zkatdlog/nogh/v1/transfer).
The code that assembles the transfer action can be found here: [`transfer.go`](./../../token/core/zkatdlog/nogh/v1/transfer.go).
The protobuf messages for the action can be found under ['nogh/protos'](./../../token/core/zkatdlog/nogh/protos/noghactions.proto).
