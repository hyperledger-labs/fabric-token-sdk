# Driver API

The **Driver API** serves as the interface bridging the generic Token API with specific token implementations. It defines the protocols for token creation, transfer, and management within a given system.

Each driver must implement the `driver.Driver` interface, fulfilling three primary objectives:
1.  **Public Parameters Discovery**: Decodes raw bytes into driver-specific public parameters.
2.  **TMS Instantiation**: Provides a mechanism to instantiate a new `TokenManagementService` (TMS) tailored to the driver and its public parameters.
3.  **Default Validation**: Returns a default `Validator` instance configured with the driver's public parameters.

## Core Architecture

The `Token Management Service` interface, implemented by the driver, functions as the core execution engine for a specific TMS instance (defined by its network, channel, and namespace).

```mermaid
graph TD
    subgraph "Core Architecture"
        Driver -- creates --> TokenManagerService
        TokenManagerService -- "provides access to" --> SpecializedServices["Specialized Services (Issue, Transfer, etc.)"]
        TokenManagerService -- "manages" --> PublicParameters
    end
```

It provides access to several specialized services that handle different aspects of the token lifecycle.

## Token Operations

The lifecycle of a token (issuance, transfer, upgrade) is managed by the following services:

```mermaid
graph TD
    subgraph "Token Operations"
        IssueService
        TransferService
        TokensService
        TokensUpgradeService
    end
```

*   **Issue Service**: Orchestrates the issuance of new tokens. It generates `IssueAction` and `IssueMetadata`, allowing authorized parties to create tokens and assign them to recipients. It also handles the verification and deserialization of issuance actions.
*   **Transfer Service**: Manages the transfer of token ownership. It generates `TransferAction` and `TransferMetadata`, enabling the movement of tokens from one party to another while ensuring transaction integrity. It also handles the verification and deserialization of transfer actions.
*   **Tokens Service**: Provides general token management utilities, such as de-obfuscating token outputs to reveal their details (type, value, owner, format) and extracting recipient identities. It also reports the token formats supported by the driver.
*   **Tokens Upgrade Service**: Manages the token upgrade lifecycle (e.g., from FabToken to ZKAT-DLog). It generates upgrade challenges, produces zero-knowledge proofs for tokens being upgraded, and verifies these proofs.

## Validation & Auditing

The driver provides mechanisms to ensure that token transactions are valid and compliant with the system's rules.

```mermaid
graph TD
    subgraph "Validation & Auditing"
        Validator
        AuditorService
        CertificationService
        PublicParamsManager
        Authorization
    end
```

*   **Validator**: Performs rigorous validation of token transactions. It unmarshals actions from a token request and verifies the request against the ledger state and a provided anchor.
*   **Auditor Service**: Enables auditing capabilities, allowing authorized auditors to inspect token requests and their associated metadata against a provided anchor to ensure compliance.
*   **Certification Service**: Manages token certifications, providing mechanisms for certifying tokens on the ledger and verifying their authenticity.
*   **Public Parameters Manager**: Manages the TMS instance's public parameters, providing access to the parameters themselves, their hash, and facilitating the generation of certifier key pairs.
*   **Authorization**: Checks the relationships between tokens and wallets. It determines if a token belongs to a known owner wallet, if the service has auditor privileges, and verifies if an issuer authorized a specific token.

## Identity & Wallet Management

The management of cryptographic identities and wallets is handled by the following services:

```mermaid
graph TD
    subgraph "Identity & Wallet Management"
        WalletService -- provides --> OwnerWallet
        IdentityProvider
        Deserializer
        Configuration
    end
```

*   **Wallet Service**: Handles the management of different types of wallets (issuer, owner, auditor, certifier). It facilitates wallet lookup, identity registration, and provides access to wallet-specific functionalities like balance calculation and token listing.
*   **Identity Provider**: Acts as a central registry for identities and their cryptographic materials. It manages the registration and retrieval of signature signers and verifiers, audit information, and enrollment IDs.
*   **Deserializer**: Responsible for converting serialized identity data into cryptographic verifiers and extract recipient identities. It also provides matchers for audit information.
*   **Configuration**: Provides access to TMS-specific configuration settings (e.g., paths, identifiers), allowing the services to adapt their behavior based on the environment.

Currently, the Fabric Token SDK offers two reference driver implementations: `FabToken` and `ZKAT-DLog` (Zero-Knowledge Authenticated Token based on Discrete Logarithm).

## Serialization

The Driver API relies on the **Protocol Buffers (Protobuf)** protocol for all serialized data structures. This ensures consistent communication between nodes and the ledger while guaranteeing backward and forward compatibility.

### Public Parameters
Public parameters are wrapped in a generic envelope that carries:
- **Identifier**: A unique string identifying the driver and version (e.g., `zkatdlognogh/v1`).
- **Raw Bytes**: The driver-specific serialization of the parameters, which only the corresponding driver can decode.

### Token Requests
A `TokenRequest` is the primary structure submitted to the ledger. It consists of:
- **Actions**: A list of serialized `IssueAction` or `TransferAction` objects.
- **Signatures**: Digital signatures from the parties authorizing the actions (owners, issuers).
- **Auditing**: A dedicated section for auditor signatures and their identities.
- **Metadata**: (Sent via transient data) Contains cleartext information required by participants and auditors to process the transaction.

## Drivers

The Token SDK comes equipped with two reference drivers:

- [**FabToken**](./drivers/fabtoken.md): A straightforward implementation prioritizing simplicity. It stores token transaction details (type, value, owner) in cleartext on the ledger, using X.509 certificates for identities.
- [**DLOG w/o Graph Hiding (NOGH)**](./drivers/dlogwogh.md): A privacy-preserving driver using Zero-Knowledge Proofs (ZKP) to hide token types and values via Pedersen commitments. It leverages Idemix for owner anonymity while revealing the spending graph.
