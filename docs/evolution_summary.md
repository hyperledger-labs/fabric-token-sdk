# Evolution of Fabric Token SDK (v0.4.0 -> Present)

Since tag `v0.4.0` (January 2025), the Fabric Token SDK has undergone a significant architectural and functional evolution, comprising over 700 commits. The evolution can be summarized in several key areas:

## Architectural Refinement: Common Driver Framework
The SDK has transitioned toward a more modular and robust internal driver architecture. A major shift involved moving core driver interfaces and shared logic into the `token/core/common/driver` package. This decoupling allows for more uniform driver implementations and simplifies the introduction of new token technologies.
*   *See also:* [**Driver API Overview**](driverapi.md)

## Privacy-Preserving Driver Evolution: `zkatdlog/nogh`
One of the most substantial additions is the `nogh` (**No Graph Hiding**) variant of the `zkatdlog` driver. This variant represents an optimization and refinement of the zero-knowledge token logic, providing a more efficient way to handle private transactions by omitting graph-hiding properties. 
*   *See also:* [**DLog w/o Graph Hiding (NOGH) Driver**](drivers/dlogwogh.md)

- **Upgradability Support:** The `nogh` driver explicitly supports token upgradability. 
    - **In-place Upgrades:** Compatible legacy tokens (like Fabtoken) can be spent directly by the `nogh` driver. The driver automatically generates Pedersen commitments and an "Upgrade Witness" on the fly, allowing for transparent migration during standard transfer operations. Criteria for compatibility include format inclusion in the driver's supported list and precision alignment (e.g., legacy precision $\le$ current max precision).
    - **Burn and Re-issue:** For major migrations, it implements a "Burn and Re-issue" mechanism. This is facilitated by a dedicated `upgrade` service that handles challenge-response protocols and proof verification to ensure supply consistency during migration.
    *   *See also:* [**Upgradability Guide**](upgradability.md)
- **Data Model & Protocol Buffers:** The `nogh` driver introduces a refined data model for tokens and metadata. Tokens are represented as Pedersen commitments to the token type, value, and blinding factor. Newly defined protocol buffers support this optimized model.
    *   *See also:* [**DLog Protobuf Messages**](drivers/dlogwogh.md#protobuf-messages)
- **Driver Versioning:** The implementation follows a clear versioning strategy, with the current `nogh` driver logic residing in the `v1` sub-package (`token/core/zkatdlog/nogh/v1`). This versioning ensures that future protocol changes can be introduced without breaking backward compatibility for existing ledger data.
- **Crypto & Math:** New `bls12_381_bbs` curve support and improved cryptographic primitives are integrated into the `v1` implementation.
- **Auditing:** A dedicated auditing layer (`v1/audit`) provides necessary compliance tools for private token systems.

## Token Transaction (TTX) Service Modernization
The Token Transaction service, a core component for orchestrating token lifecycles, has been significantly refactored:
- **Dependency Injection Pattern:** The introduction of the `dep/` sub-package in the TTX service enables better dependency management, reducing circular dependencies and drastically improving unit testability.
- **Multisig Support:** New capabilities for multi-signature transactions have been introduced in `token/services/ttx/multisig`.
- **Finality Tracking:** Improved transaction finality handling via the new `token/services/ttx/finality` package, including a new `TTXRecoveryHandler` for synchronous transaction recovery.
- **Transaction Recovery:** A new recovery mechanism has been introduced to automatically re-register finality listeners for pending transactions that may have lost their listeners due to node restarts or network issues.
*   *See also:* [**TTX Service Documentation**](services/ttx.md)

## Core Service Enhancements
The SDK's foundational services have been matured:
- **Identity Service:** Reorganized to better handle various identity types, with improved support for Idemix (privacy-preserving) and X.509 (standard) identities.
    *   *See also:* [**Identity Service**](services/identity.md)
- **Storage Service:** Enhanced with more efficient and reliable ways to manage token and transaction databases.
    *   *See also:* [**Storage Service**](services/storage.md)
- **Network Service:** Expanded to handle more complex Fabric network interactions and better integration with the Fabric Smart Client. A major change was the **removal of the Orion-based implementation**, which has been replaced by the introduction of **FabricX support**, providing a more modern and integrated approach for advanced ledger interactions.
    *   *See also:* [**Network Service**](services/network.md)

## Developer Experience: Testing & Documentation
The SDK has seen a major push in both testing infrastructure and documentation:
- **Comprehensive Mocks:** A massive addition of mocks (`token/core/common/driver/mock`, etc.) simplifies application testing and improves SDK maintainability.
- **Expanded Documentation:** The `docs/` directory has been substantially populated with detailed guides on driver APIs, token SDK usage, architectural diagrams, and development best practices.
- **Integration Tests:** The `integration/` tests have been expanded to cover new use cases, ensuring the stability of the evolved features.
    *   *See also:* [**Testing Guide**](development/testing.md)

## High-Level Token API Evolution
The developer-facing Token API has been refined for better usability, consistency, and cross-version stability:
-   **Ubiquitous Context Support:** Almost all API methods (in `WalletManager`, `Vault`, `TMS`, etc.) now accept a `context.Context`. This enables standard Go practices for cancellation, timeouts, and request-scoped tracing.
-   **Unified Option Pattern:** The introduction of a centralized `opts.go` and the functional options pattern (e.g., `WithTMSID`, `WithDuration`) provides a consistent and extensible way to configure API calls.
-   **New `TokensService`:** A high-level service for advanced token operations, including de-obfuscation and support for the new in-place upgrade challenge/proof protocol.
-   **Robust TMS Management:** The `ManagementService` now performs more thorough eager initialization of its sub-services and supports dynamic cache clearing via the `Update` mechanism, facilitating seamless public parameter rotations.
-   **Enhanced Querying:** The `Vault` and `QueryEngine` have been expanded with new methods like `GetTokenOutputs` and `WhoDeletedTokens`, providing deeper insights into the token lifecycle on the ledger.
-   **Protobuf Backed Requests:** The high-level `Request` structures now leverage versioned Protobuf messages internally, ensuring that transaction blueprints created by one node version can be understood by others.
*   *See also:* [**Token API Documentation**](tokenapi.md)

## Conclusion
The evolution since `v0.4.0` marks the SDK's transition into a mature, modular, and feature-rich framework. The focus on architectural clarity, optimized privacy-preserving drivers, and a better developer experience through documentation and testability has solidified its role as a key tool for building tokenized applications on Hyperledger Fabric.
