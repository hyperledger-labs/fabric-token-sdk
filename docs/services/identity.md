# Identity Service

The **Identity Service** (`token/services/identity`) is an **internal infrastructure service** of the Fabric Token SDK. It provides a unified interface for managing identities, signatures, and verification, operating **independently** of the core Fabric Smart Client (FSC) identity service. 

This independence ensures that token-related cryptographic material (such as Idemix pseudonyms or X.509 certificates used for token ownership) is managed according to the specific privacy and security requirements of the Token Drivers, regardless of the underlying DLT platform.

## Overview

The Identity Service abstracts the complexity of different cryptographic schemes, allowing the SDK to support multiple identity types (e.g., X.509, Idemix) and different storage backends seamlessly. 

It is a fundamental component used by token drivers and application services (like the TTX service) to handle:
*   **Signature Management**: Generating and verifying signatures for token requests.
*   **Identity Resolution**: Resolving long-term identities to ephemeral pseudonyms and vice-versa.
*   **Auditability**: Managing audit information to reveal the enrollment ID behind an anonymous identity (when authorized).
*   **Wallet Management**: Handling identities for different roles such as Issuer, Auditor, Owner, and Certifier.

## Architecture

The Identity Service implements the **Driver API** interfaces defined in `token/driver/wallet.go`. This ensures that the Token Management System (TMS) can interact with any identity implementation through a standard set of methods.

### Component Mapping

The following table shows how the internal components map to the Driver API interfaces:

| Component              | Implements Driver Interface | Description                                           |
|:-----------------------|:----------------------------|:------------------------------------------------------|
| `identity.Provider`    | `driver.IdentityProvider`   | Core identity management & verification.              |
| `wallet.Service`       | `driver.WalletService`      | Registry for all wallets (Owner, Issuer, etc.).       |
| `role.LongTermOwnerWallet` | `driver.OwnerWallet`      | Long-Term Identity-based Owner wallet functionality.  |
| `role.AnonymousOwnerWallet` | `driver.OwnerWallet`     | Anonymous Identity-based Owner wallet functionality.  |
| `role.IssuerWallet`    | `driver.IssuerWallet`       | Issuer wallet functionality.                          |
| `role.AuditorWallet`   | `driver.AuditorWallet`      | Auditor wallet functionality.                         |
| `role.CertifierWallet` | `driver.CertifierWallet`    | Certifier wallet functionality.                       |

### Component Interaction

```mermaid
classDiagram
    direction TB
%% Driver Interfaces
    class IdentityProvider {
        <<interface>>
        +GetSigner()
        +GetAuditInfo()
        +IsMe()
    }
    class WalletService {
        <<interface>>
        +OwnerWallet()
        +IssuerWallet()
        +RegisterRecipientIdentity()
    }

%% Concrete Implementations
    class identity_Provider["identity.Provider"] {
        -Storage
        -Deserializers
        -SignerCache
    }
    class wallet_Service["wallet.Service"] {
        -RoleRegistry
        -IdentityProvider
        -OwnerWallet
        -IssuerWallet
        -AuditorWallet
        -CertifierWallet
    }
    class role_Role["role.Role"] {
        -LocalMembership
        +GetIdentityInfo()
    }
    class membership_KeyManagerProvider["membership.KeyManagerProvider"] {
        <<interface>>
        +Get() KeyManager
    }

    identity_Provider ..|> IdentityProvider : Implements
    wallet_Service ..|> WalletService : Implements
    wallet_Service --> identity_Provider : Uses
    wallet_Service --> role_Role : Uses (via RoleRegistry)
    role_Role --> membership_KeyManagerProvider : Uses (via LocalMembership)

    note for membership_KeyManagerProvider "Handles low-level crypto<br/>and identity verification"
    note for wallet_Service "High-level management<br/>of wallets and roles"
```

### LocalMembership

The `LocalMembership` component (`token/services/identity/membership`) plays a pivotal role in managing local identities for a specific role (e.g., Owner, Issuer).

*   **Binding**: Each instance is bound to a list of **Key Managers**.
*   **Identity Wrapping**: When a Key Manager generates an identity (based on the configuration), `LocalMembership` automatically wraps it using `WrapWithType`. 
    This ensures that the generated identity carries the correct type information required by the system (as defined in `token/services/identity/typed.go`).
*   **Role Implementation**: `LocalMembership` serves as the foundational implementation for `role.Role`. 
    When you interact with a Role to resolve an identity or sign a transaction, you are effectively delegating to the underlying `LocalMembership`.

### Example: Wiring Services

The following example demonstrates how these services are instantiated and wired together, as seen in the ZKATDLog driver:

```go
func (d *Base) NewWalletService(...) (*wallet.Service, error) {
    // 1. Create Identity Provider
    identityProvider := identity.NewProvider(...)

    // 2. Initialize Membership Role Factory
    roleFactory := membership.NewRoleFactory(...)

    // 3. Configure Key Managers (e.g. Idemix and X.509 for Owner role)
    // we have one key manager to handle fabtoken tokens and one for each idemix issuer public key in the public parameters
    kmps := make([]membership.KeyManagerProvider, 0)
    // ... add Idemix Key Manager Providers ...
    kmps = append(kmps, x509.NewKeyManagerProvider(...))

    // 4. Create and Register Roles
    roles := role.NewRoles()
    
    // Owner Role (with anonymous identities)
    ownerRole, err := roleFactory.NewRole(identity.OwnerRole, true, nil, kmps...)
    roles.Register(identity.OwnerRole, ownerRole)
    
    // Issuer Role (no anonymous identities)
    issuerRole, err := roleFactory.NewRole(identity.IssuerRole, false, pp.Issuers(), x509.NewKeyManagerProvider(...))
    roles.Register(identity.IssuerRole, issuerRole)
    
    // ... Register Auditor and Certifier roles ...

    // 5. Create Wallet Service with the registered roles
    return wallet.NewService(
        logger,
        identityProvider,
        deserializer,
        // Convert the roles registry into the format expected by the wallet service
        wallet.Convert(roles.Registries(...)),
    ), nil
}
```

## Identity Types

The Identity Service leverages a wrapper called **TypedIdentity** to support various identity schemes uniformly. 
This allows the SDK to be extensible and capable of handling different cryptographic requirements.

### TypedIdentity

`TypedIdentity` (defined in `token/services/identity/typed.go`) acts as a generic container. 
It wraps the raw identity bytes with a type label, enabling the system to verify deserializers and process signatures correctly without hardcoding implementation details.

*   **Encoding**: ASN.1 encoded `SEQUENCE`.
*   **Structure**:
    - `Type` (string): The identifier of the identity scheme (e.g., `"x509"`, `"idemix"`).
    - `Identity` (bytes): The raw payload of the identity, specific to the key manager.

### Default Key Managers

The identity service includes two primary implementations for concrete identities:

#### 1. X.509
Standard PKIX identities.
*   **Identity (Payload)**: A standard X.509 certificate.
*   **Audit Info**: JSON-encoded `AuditInfo` structure containing the Enrollment ID and Revocation Handle.
    - `EID` (string): The enrollment identifier.
    - `RH` (bytes): The revocation handle.
*   **Encoding**:
    - `TypedIdentity` payload: Raw X.509 certificate bytes.
    - Audit Info: JSON.
*   **Usage**: Ideal for infrastructure components (nodes, services) or scenarios where anonymity is not required.
*   **Implementation**: `token/services/identity/x509`.

#### 2. Idemix (Identity Mixer)
Advanced identity encryption based on Zero-Knowledge Proofs (ZKP).
*   **Identity (Payload)**: A **full Idemix signature** acting as a commitment to the user's attributes. It is encoded as a Protobuf `SerializedIdemixIdentity` message.
    - `NymPublicKey` (bytes): The pseudonym public key ($N = g^{sk} \cdot h^r$).
    - `Proof` (bytes): A zero-knowledge proof of credential possession and nym derivation.
    - `Schema` (string): The version of the credential schema.
*   **Audit Info**: JSON-encoded `AuditInfo` structure.
    - `EidNymAuditData`: Cryptographic data required to de-anonymize the Enrollment ID.
    - `RhNymAuditData`: Cryptographic data required to de-anonymize the Revocation Handle.
    - `Attributes` (array of bytes): The cleartext values of the attributes (e.g., EID at index 2, RH at index 3).
    - `Schema` (string): The credential schema version.
*   **Encoding**:
    - `TypedIdentity` payload: Protobuf.
    - Audit Info: JSON.
*   **Anonymity**: Users can prove they hold a valid credential without revealing their actual identity.
*   **Unlinkability**: Different transactions from the same user appear uncorrelated.
*   **Auditability**: Authorized auditors can reveal the Enrollment ID using the audit info.
*   **Signature Format**: Signatures are **nym signatures** (pseudonym-based) that do not carry attributes, providing unlinkability between transactions.
*   **Implementation**: `token/services/identity/idemix`.

#### 3. IdemixNym (Idemix with Pseudonym-based Identity)
An extension of Idemix that uses a **commitment to the Enrollment ID (EID)** as the identity instead of the full Idemix signature.
*   **Identity (Payload)**: A small **Nym EID** (a cryptographic commitment to the enrollment ID, $g^{sk} \cdot h^{r_{eid}}$).
*   **Audit Info**: JSON-encoded structure that extends the standard Idemix `AuditInfo`.
    - Includes all fields from Idemix `AuditInfo`.
    - `IdemixSignature` (bytes): The full Idemix signature that would have been the identity in the standard Idemix manager.
*   **Encoding**:
    - `TypedIdentity` payload: Raw bytes of the nym.
    - Audit Info: JSON.
*   **Signature Packaging**: Signatures are wrapped in an ASN.1 `SEQUENCE` containing:
    - `Creator` (bytes): The full Idemix signature (enabling verification against the IPK).
    - `Signature` (bytes): The actual pseudonym signature bytes.
*   **Enhanced Privacy**: The identity itself is a pseudonym (nym) rather than the full Idemix signature with attributes.
*   **Reduced Identity Size**: The nym EID is significantly smaller than a full Idemix signature, reducing storage and transmission overhead.
*   **Backward Compatible Auditability**: Maintains full auditability through the audit info, which contains both the nym proof and the original Idemix signature.
*   **Implementation**: `token/services/identity/idemixnym`.

**Key Differences from Standard Idemix:**

| Aspect | Idemix | IdemixNym |
|:-------|:-------|:----------|
| **Identity (Token Owner)** | Full Idemix signature with attributes | Nym EID (commitment to enrollment ID) |
| **Identity Payload Encoding** | Protobuf | Raw bytes |
| **Audit Info Encoding** | JSON | JSON (extended) |
| **Signature Encoding** | Raw bytes | ASN.1 (Creator + Signature) |
| **Identity Size** | Large (~several KB) | Small (~32-64 bytes) |
| **Storage Overhead** | High | Low |

### Other Identity Types

The architecture supports specialized identity types for complex use cases:

#### Multisig
Located in `token/services/identity/multisig`.
*   **Concept**: An identity that wraps multiple sub-identities.
*   **Identity (Payload)**: An ASN.1 encoded `MultiIdentity` sequence.
    - `Identities` (array of `TypedIdentity` bytes): The constituent identities.
*   **Audit Info**: JSON-encoded `AuditInfo` structure.
    - `IdentityAuditInfos` (array of `IdentityAuditInfo`): A list of audit information blobs for each constituent identity.
*   **Encoding**:
    - `TypedIdentity` payload: ASN.1.
    - Audit Info: JSON.
*   **Usage**: Useful for requiring multiple signatures or representing a group of parties.
*   **Auditability**: Aggregates audit information for all underlying identities.

#### HTLC (Hashed Time Lock Contract)
Located in `token/services/identity/interop/htlc`.
*   **Concept**: A script-based identity used primarily for interoperability mechanisms like atomic swaps.
*   **Identity (Payload)**: A JSON-encoded `Script` structure defining the swap conditions.
    - `Sender` (bytes): The wrapped identity of the sender.
    - `Recipient` (bytes): The wrapped identity of the recipient.
    - `Deadline` (uint64): The timeout period.
    - `HashInfo`: Information about the hash lock.
*   **Audit Info**: A JSON-encoded `ScriptInfo` structure.
    - `Sender` (bytes): The audit info for the sender's identity.
    - `Recipient` (bytes): The audit info for the recipient's identity.
*   **Encoding**:
    - `TypedIdentity` payload: JSON.
    - Audit Info: JSON.
*   **Behavior**: Validation involves satisfying the script conditions (e.g., providing the hash preimage).

## Extending the Identity Service

The Identity Service is designed to be extensible through the driver interfaces
defined in the token SDK. Custom identity implementations can be provided by
implementing the required identity and wallet interfaces.

Typical extension scenarios include:
- Supporting a new identity type by implementing a custom `KeyManager`
- Customizing signature generation or verification logic within a `KeyManager`
- Providing a custom `KeyManagerProvider` to plug new identity mechanisms into `LocalMembership`
