## FabToken: A Simple Token Driver

FabToken is a straightforward implementation of the Driver API. 
It prioritizes simplicity over privacy, storing all token transaction details openly on the ledger for anyone with access to view ownership and activity.

### Configuration

FabToken recognizes [`public parameters`](../../token/core/fabtoken/setup.go) containing the following information:

* **Label:** A unique identifier associated with the configuration, often used for versioning.
* **Quantity Precision:** Defines the level of detail used to represent token amounts.
* **Auditor (Optional):** If set, specifies the identity of an authorized auditor who can approve token requests.
* **Issuers:** A list of authorized issuers who can create new tokens.
* **MaxToken:** The maximum quantity a token can hold.

**Important:** The `Label` field must be set to `"fabtoken"`. This driver supports multiple issuers but only one auditor (if enabled).

### Supported Identities

FabToken exclusively supports long-term identities based on a standard X.509 certificate scheme. 
These identities contain an X.509 certificate, which reveals the owner's enrollment ID in plain text.

**Public Parameter Requirements:** Both `Auditor` (optional) and `Issuers` fields within the public parameters must contain serialized X.509-based identities.

### Managing Wallets

The wallet service provides access to a party's various token wallets. Each party can have multiple wallets for different purposes. However, each wallet is linked to a single X.509-based identity.

### Tokens on the Ledger

Tokens are directly represented on the ledger as JSON-formatted data based on the [`token.Token`](../../token/token/token.go) structure. 
The `Owner` field of this structure stores the identity information. 
The [`Identity Service`](../services/identity.md) handles the encoding/decoding of this field.

### Ensuring Secure Transactions

FabToken's validation process enforces several critical security measures:

* **Authorized Issuance:** Only issuers whose identities are registered in the public parameter's `Issuers` field can create tokens. If this list is empty, anyone can issue tokens (not recommended for production).
* **Ownership Verification:** Only the legitimate owner of a token can transfer it.
* **Balanced Transfers:** In a transfer transaction, the total value of tokens being transferred in (inputs) must equal the total value being transferred out (outputs).
* **Redemption Control:** Only the owner of a token can redeem it.
* **Optional Auditing:** If an auditor is specified in the public parameters, their signature is required on all token requests for them to be valid.

This revised version removes references to Fabric and emphasizes FabToken's compatibility with various blockchain backends.