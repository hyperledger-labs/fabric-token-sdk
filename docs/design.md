# The Fabric Token SDK

**What it is:**

* A set of APIs and services for building token-based applications on Hyperledger Fabric, Orion, and potentially other platforms.

**Key Features:**

* Uses the `Unspent Transaction Output` (UTXO) model for tracking token movements.
* Manages cryptographic keys through `Wallets`, keeping track of owned unspent outputs.
* Supports `various privacy levels`, from fully transparent to Zero-Knowledge Proofs for obfuscating transaction details.
* Allows developers to create `custom services` on top of the core API for specific application needs.

**Architecture:**

* The Fabric Token SDK stack consists of several layers:
  * [`Services`](services/services.md): Pre-built functionalities like assembling transactions and selecting unspent tokens.
  * [`Token API`](apis/token-api.md): Provides a common abstraction for interacting with tokens across different backends.
  * [`Driver API`](apis/driver-api.md): Translates generic token operations into backend-specific calls (e.g., Fabric vs. Orion).
  * [`Drivers`](drivers/drivers.md): Define token representation, operations, and validation rules for specific implementations.

**Additional Information:**

* The SDK leverages the Fabric Smart Client stack for complex workflows, secure storage, and event listening.
* Configuration examples can be found in the [`Example Core File Section`](./core-token.md) documentation.
* Deployment guideline can be found in the [`Deployment`](./deployment/deployment.md) documentation.