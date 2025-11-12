# The Fabric Token-SDK

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents** 

- [Introduction](#introduction)
- [Consumer Interaction Model](#consumer-interaction-model)
- [Requirements and Use Cases](#requirements-and-use-cases)
- [Prerequisites, Dependencies, Dependents, and Incompatibilities](#prerequisites-dependencies-dependents-and-incompatibilities)
- [Terminology and Glossary](#terminology-and-glossary)
- [Architecture, Interfaces, and Impact](#architecture-interfaces-and-impact)
- [Developer Experience](#developer-experience)
- [Command Line Interface (CLI)](#command-line-interface-cli)
- [Performance, Scalability, and Resource Consumption](#performance-scalability-and-resource-consumption)
- [Serviceability, Logging and Troubleshooting](#serviceability-logging-and-troubleshooting)
- [Monitoring, Metrics and Events](#monitoring-metrics-and-events)
- [High Availability and Disaster Recovery](#high-availability-and-disaster-recovery)
- [Build, Packaging and Deployment](#build-packaging-and-deployment)
- [Platform Support](#platform-support)
- [Testing](#testing)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## Introduction

The Fabric Token-SDK (FTS, for short) is set of APIs and services for building token-based applications on Hyperledger Fabric, and potentially other platforms.

The key Features are:
* Uses the `Unspent Transaction Output` (UTXO) model for tracking token movements.
* Manages cryptographic keys through `Wallets`, keeping track of owned unspent outputs.
* Supports `various privacy levels`, from fully transparent to Zero-Knowledge Proofs for obfuscating transaction details.
* Allows developers to create `custom services` on top of the core API for specific application needs.

The Fabric Token SDK stack consists of several layers:
* Services: Pre-built functionalities like assembling transactions and selecting unspent tokens.
* Token API: Provides a common abstraction for interacting with tokens across different backends.
* Driver API: Translates generic token operations into backend-specific calls (e.g., Fabric).
* Drivers: Define token representation, operations, and validation rules for specific implementations.

The SDK leverages the Fabric Smart Client stack for complex workflows, secure storage, and event listening.

The Fabric Token SDK has evolved beyond its initial focus on Hyperledger Fabric.
It now empowers developers with the following capabilities across various platforms, including permissioned blockchains like Fabric:

* **Tokenization Made Easy:** Create tokens representing any type of asset, be it physical or digital.
* **Privacy by Design:** Select the appropriate privacy level for your specific use case, without modifying your application logic.
* **Peer-to-Peer Transactions:** Orchestrate token transfers directly between users, streamlining the process.
* **Atomic Swaps:** Facilitate secure exchanges of different tokens without relying on intermediaries.
* **Transaction Auditing:** Review transactions before they are finalized, ensuring accuracy and compliance.
* **Interoperability:** Connect with token systems on other blockchain networks, fostering broader ecosystems.
* **Seamless Integration:** Add a token layer to existing applications, regardless of platform, with minimal effort.

## Consumer Interaction Model

The Token-SDK APIs are consumed by developers to perform various token-related operations.
The developers must decide how to configure the Token-SDK to achieve the intended goals.
Moreover, they are also responsible for defining the initial content of the datasource used by the Token-SDK.

## Requirements and Use Cases

Token-based applications that require:
- Privacy;
- Bankend agnostic;

## Prerequisites, Dependencies, Dependents, and Incompatibilities

The SDK leverages the following related projects:
- [`Fabric Smart Client (FSC, for short)`](https://github.com/hyperledger-labs/fabric-smart-client): For complex workflows orchestration, secure storage, and event listening.
- [`Idemix`](https://github.com/IBM/idemix): For the anonymous credentials.
- [`Mathlib`](https://github.com/IBM/mathlib): For the elliptic curve math operations.

The system administrator is responsible for preparing:

- `Configuration`: FTS needs to be configured accordingly to the specific use-case.
  FTS uses the FSC's `config service` to access its configuration.
  An example of such a configuration can be found [`here`](./core-token.md).
- `Data storage`: FTS requires an SQL data source to store relevant information for its functioning.
- `HSM`: When relevant to store secret keys.
- `External Key Store`: When relevant to store secret keys

In some configurations, FTS might run in a container that is volume-less.

Other pre-requisites come from the Fabric Smart Client directly.

## Terminology and Glossary

- For an introduction into the concepts of `Database`, `Persistence`, `Driver`, `Store`, read [this documentation](https://github.com/hyperledger-labs/fabric-smart-client/blob/main/docs/platform/view/db-driver.md).
- FTS stands for Fabric Token SDK
- FSC stands for Fabric Smart Client

## Architecture, Interfaces, and Impact

The FTS stack is summarized by the following diagram:

![image](imgs/stack.png)

It consists of the following layers:

* [`Services`](./services.md): Pre-built functionalities: Assembling transactions, selecting unspent tokens, and so on.
* [`Token API`](./tokenapi.md): Provides a common abstraction for interacting with tokens across different backends.
* [`Driver API`](./driverapi.md): These are the API on top of which the `Token API` is built. The Driver API is instantiated in a `Driver`.
* [`Drivers`](./driverapi.md): A `Driver` implements the `Driver API` and defines token representation, operations, and validation rules.

## Developer Experience

The developers will face both the Token API and the services to build their own token-based applications.
Driver developers will have to face instead the Driver API.

## Command Line Interface (CLI)

FTS comes equipped with `tokengen`. It is a utility for generating Fabric Token-SDK material.
It is provided as a mean of preconfiguring public parameters, token chaincode, and so on.

For a complete list of available commands, see this [`page`](./../cmd/tokengen/README.md).

## Performance, Scalability, and Resource Consumption

The Token-SDK handles the entire lifecycle of a token transactions.
Different parts of the Token-SDK will run on different network nodes with different roles.
The careful orchestration of their interactions guarantees that a token transaction is successfully processed.

## Serviceability, Logging and Troubleshooting

The Token-SDK uses the logging infrastructure offered by the Fabric Smart Client.

## Monitoring, Metrics and Events

The Token-SDK adopts the monitoring infrastructure provided by the [`Fabric Smart Client`](https://github.com/hyperledger-labs/fabric-smart-client/blob/main/docs/platform/view/monitoring.md).

We use the following two methods to monitor the performance of the application:
* **Metrics** provide an overview of the overall system performance using aggregated results, e.g. total requests, requests per second, current state of a variable, average duration, percentile of duration
* **Traces** help us analyze single requests by breaking down their lifecycles into smaller components

## High Availability and Disaster Recovery

The Fabric Token SDK's design allows multiple replica nodes to be attached to the same shared datasource.
If conflict arises, then only one replica will succeed.
Replica nodes can be attached to the shared datasource on-demand.
This way, the new replica is aware of the current status of all transactions and tokens processed so far.

## Build, Packaging and Deployment

The Token-SDK is embedded as a dependency in a third-party application.

## Platform Support

The Token-SDK is written in Go. Therefore, any platform supporting Go can run it.
We require at least `go 1.24.2`.

Certain components might require CGO (e.g. the HSM support)

## Testing

The Token-SDK come equipped with unit and integration tests.
As for the Fabric Smart Client, the Token-SDK adopts the following philosophy [`Write tests. Not too many. Mostly integration.`](https://kentcdodds.com/blog/write-tests)