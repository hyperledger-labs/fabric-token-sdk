# Identity Service

The **Identity Service** (`token/services/identity`) provides a unified interface for managing identities, signatures, and verification within the Fabric Token SDK. It abstracts the underlying cryptographic details, allowing the SDK to support multiple identity types (e.g., X.509, Idemix) and different storage backends seamlessly.

This service is a fundamental component used by token drivers and other services (like the Token Transaction (TTX) service) to handle:
*   **Signatures**: Generating and verifying signatures for transactions.
*   **Identity Resolution**: resolving long-term identities to ephemeral identities and vice-versa.
*   **Auditability**: Managing audit information to reveal the enrollment ID behind an anonymous identity (if allowed).
*   **Role Management**: Handling identities for different roles (Issuer, Auditor, Owner, Certifier).

## Architecture

The core of the service is the `Provider` struct, which implements the `driver.IdentityProvider` interface. It orchestrates interactions between:

1.  **Storage**: Persists identity data, including audit information, secrets (via keystore), and bindings between wallets and identities.
2.  **Deserializers**: Parses raw identity bytes into typed identity structures (e.g., converting an Idemix byte array into a usable object).
3.  **Signers and Verifiers**: Interfaces for cryptographic operations.
4.  **Binder**: Handles binding long-term identities to ephemeral ones.

### Key Components

*   **`Provider`** (`token/services/identity/provider.go`): The main entry point. It manages caches for signers and "is-me" checks to optimize performance.
*   **`Storage`** (`token/services/identity/driver/storage.go`): Interface for the underlying storage. It handles storing and retrieving audit info, token metadata, and signer info.
*   **`Role`** (`token/services/identity/driver/role.go`): Defines abstract roles (Issuer, Auditor, etc.) and how they map to long-term identities.

## Identity Types

The service supports multiple identity types, extensible via drivers. The two primary built-in types are:

### 1. X.509
Standard PKIX identities.
*   **Transparency**: Identity is known.
*   **Usage**: Typically used for component identification (e.g., orderingers, peers) or when anonymity is not required.
*   **Implementation**: Located in `token/services/identity/x509`.

### 2. Idemix (Identity Mixer)
Zero-Knowledge Proof (ZKP) based identities.
*   **Anonymity**: Allows users to prove possession of a credential without revealing the credential itself.
*   **Unlinkability**: Transactions cannot be linked to the same user.
*   **Auditability**: Special "audit info" allows authorized auditors to reveal the identity if needed.
*   **Implementation**: Located in `token/services/identity/idemix`.

## Usage

### Retrieving the Identity Provider
The Identity Provider is usually accessed via the `Token Management Service` (TMS) API or injected into other services.

### Getting a Signer
To sign a message with a specific identity:

```go
// identity is of type driver.Identity (byte array)
signer, err := identityProvider.GetSigner(ctx, identity)
if err != nil {
    return err
}

signature, err := signer.Sign(message)
```

### Verifying a Signature
Unless you have the `Verifier` directly, verification often happens implicitly during transaction processing or via the `Deserializer` if you need to manually verify:

```go
verifier, err := deserializer.DeserializeVerifier(ctx, identity)
if err != nil {
    return err
}

err = verifier.Verify(message, signature)
```

### Checking "Is Me"
To check if a given identity belongs to the local node (i.e., we have the private key for it):

```go
if identityProvider.IsMe(ctx, identity) {
    // This identity is managed by this node
}
```

### Audit Information
For anonymous identities (Idemix), "Audit Information" is a crucial concept. It contains the data needed to de-anonymize the identity (e.g., to checking against a revocation list or for regulatory auditing).

```go
// Retrieve the Enrollment ID from audit info
eid, err := identityProvider.GetEnrollmentID(ctx, identity, auditInfo)
```

## Storage & Caching
The service aggressively caches:
*   **Signers**: Once deserialized/created, signers are cached to avoid expensive re-initialization.
*   **"Is Me" Checks**: Results of ownership checks are cached to speed up transaction filtering.

## Integration with Drivers
Token drivers (like the UTXO driver) heavily rely on this service. They delegate identity management tasks to it, ensuring that the driver code remains agnostic to the specific identity technology (X.509 vs Idemix) being used.
