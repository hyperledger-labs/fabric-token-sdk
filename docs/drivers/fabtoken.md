# FabToken

FabToken is a straightforward implementation of the Driver API.
It prioritizes simplicity over privacy, storing all token transaction details openly on the ledger for anyone with access to view ownership and activity.
FabToken exclusively supports long-term identities based on a standard X.509 certificate scheme,
which reveals the owner's enrollment ID in plain text.
Tokens are directly represented on the ledger as JSON-formatted data based on the `token.Token` structure.
The `Owner` field of this structure stores the identity information.
The `Identity Service` handles the encoding/decoding of this field.

## Security

`FabToken` does not provide privacy for tokens or identities.
The validator guarantees the following:
*   **Issuer Authorization**: Only issuers listed in the public parameters can issue tokens.
*   **Auditor Authorization**: Only auditors listed in the public parameters can audit transactions.
*   **Owner Authorization**: Only legitimate owners can spend their tokens.
*   **Value Preservation**: In a transfer, the sum of inputs matches the sum of outputs.
