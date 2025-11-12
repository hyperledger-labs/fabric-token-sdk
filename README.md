# Fabric Token SDK
[![License](https://img.shields.io/badge/license-Apache%202-blue)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/hyperledger-labs/fabric-token-sdk)](https://goreportcard.com/badge/github.com/hyperledger-labs/fabric-token-sdk)
[![Go](https://github.com/hyperledger-labs/fabric-token-sdk/actions/workflows/tests.yml/badge.svg)](https://github.com/hyperledger-labs/fabric-token-sdk/actions/workflows/go.yml)
[![CodeQL](https://github.com/hyperledger-labs/fabric-token-sdk/actions/workflows/codeql-analysis.yml/badge.svg)](https://github.com/hyperledger-labs/fabric-token-sdk/actions/workflows/codeql-analysis.yml)

The `Fabric Token SDK` provides a collection of APIs and services that streamline development for token-based distributed applications.

# Disclaimer

`Fabric Token SDK` has not been audited and is provided as-is, use at your own risk.
The project will be subject to rapid changes to complete the open-sourcing process, and  the list of features.

# Useful Links

- [`Documentation`](docs/tokensdk.md): The design principles of the Fabric Token SDK.
- [`Development`](docs/development/development.md): All about the development guidelines.
- [`Education Sessions`](./docs/education/README.md): Detailed education sessions with code walkthrough of the Fabric Token SDK.
- [`Fabric Samples`](https://github.com/hyperledger/fabric-samples/tree/main/token-sdk) Token SDK sample application is the
  quickest way to get a full network running with a REST API to issue, transfer and redeem tokens right away.
- [`Benchmarks`](./docs/benchmark/benchmark.md): Benchmark guidelines and reports.
- `Feedback`: Your help is the key to the success of the Fabric Token SDK. 
  - Submit your issues [`here`][`fabric-token-sdk` Issues]. 
  - Found a bug? Need help to fix an issue? You have a great idea for a new feature? Talk to us! You can reach us on
    [Discord](https://discord.gg/hyperledger) in #fabric-token-sdk.
  
- [`Fabric Smart Client`](https://github.com/hyperledger-labs/fabric-smart-client): The Token SDK leverages the 
  `Fabric Smart Client` for transaction orchestration, storing tokens and wallets, and more. Check it out.

# Getting started

Clone the code and make sure it is on your `$GOPATH`.
(Important: we assume in this documentation and default configuration that your `$GOPATH` has a single root-directory!).
Sometimes, we use `$FTS_PATH` to refer to the Fabric Token SDK repository in your filesystem.

```bash
export FTS_PATH=$GOPATH/src/github.com/hyperledger-labs/fabric-token-sdk
git clone https://github.com/hyperledger-labs/fabric-token-sdk.git $FTS_PATH
```

## Further information

Fabric Token SDK uses a system called `NWO` from Fabric Smart Client for its integration tests and samples to programmatically create a fabric network along with the fabric-smart-client nodes. The current version of fabric that is tested can be found in the project [Makefile](https://github.com/hyperledger-labs/fabric-smart-client/blob/main/Makefile) set in the `FABRIC_VERSION` variable.

In order for a fabric network to be able to be created you need to ensure you have downloaded the appropriate version of the hyperledger fabric binaries from [Fabric Releases](https://github.com/hyperledger/fabric/releases) and unpack the compressed file onto your file system. This will create a directory structure of /bin and /config. You will then need to set the environment variable `FAB_BINS` to the `bin` directory. For example if you unpacked the compressed file into `/home/name/fabric` then you would

```bash
export FAB_BINS=/home/name/fabric/bin
```

Do not store the fabric binaries within your fabric-token-sdk cloned repo as this will cause problems running the samples and integration tests as they will not be able to install chaincode.

Almost all the samples and integration tests require the fabric binaries to be downloaded and the environment variable `FAB_BINS` set to point to the directory where these binaries are stored. One way to ensure this is to execute the following in the root of the fabric-token-sdk project

```shell
make download-fabric
export FAB_BINS=$PWD/../fabric/bin
```

You can also use this to download a different version of the fabric binaries for example

```shell
FABRIC_VERSION=2.5 make download-fabric
```

If you want to provide your own versions of the fabric binaries then just set `FAB_BINS` to the directory where all the fabric binaries are stored.

# Additional Resources

- (March 17, 2022) [`Hyperledger in-Depth: Tokens in Hyperledger Fabric: What’s possible today and what’s coming`](https://www.hyperledger.org/learn/webinars/hyperledger-in-depth-tokens-in-hyperledger-fabric-whats-possible-today-and-whats-coming):
  Tokenizing the physical world is a hot blockchain topic in the industry, especially as it relates to the 
  trade of tokens as a basis of new forms of commerce. In this Hyperledger Foundation member webinar, 
  the IBM Research team describes in this webinar what tokenization use cases are possible with Hyperledger Fabric today, 
  and what enhancements are in the works (aka Fabric Token SDK).
- (October 12, 2023) [How to create a currency management app and deploy it to a Hyperledger Fabric network](https://www.youtube.com/watch?v=PX9SDva97vQ):
  In this comprehensive guide, we'll walk you through two essential aspects of the Fabric Token-SDK. Firstly, you'll learn how to develop a straightforward token application to manage a currency. You'll grasp the fundamentals of creating tokens, and implementing transaction logic using the Fabric Token-SDK. Once you've mastered the application development, we'll then show you how to effortlessly deploy it in your existing Fabric network, ensuring a seamless integration with your blockchain infrastructure. By the end of this tutorial, you'll be equipped with the skills to expand your blockchain capabilities and unleash the true potential of decentralized currency management. (Refers to [Fabric Samples](https://github.com/hyperledger/fabric-samples/tree/main/token-sdk))

# Motivation

**Hyperledger Fabric: Blockchain Built for Business**

Hyperledger Fabric ([https://hyperledger-fabric.readthedocs.io/](https://hyperledger-fabric.readthedocs.io/)) is an open-source platform designed for permissioned blockchain networks. It offers a modular and extensible architecture, allowing for customization and future growth.  Unlike traditional blockchains, Fabric applications can be written in any general-purpose programming language, making them more accessible to developers.

**Beyond Cryptocurrencies: Tokenizing the World**

While blockchain is often associated with cryptocurrencies, its potential extends far beyond. Fabric allows for the creation of tokens that represent real-world assets, both fungible (like loyalty points) and non-fungible (like unique digital artwork). This opens doors for new business models and unlocks additional value from existing assets.

**The Challenge: Building Tokenized Applications**

Developing applications that leverage tokens on Hyperledger Fabric can be complex. Fabric lacked a built-in SDK for creating and managing tokens, forcing developers to build solutions from scratch.  This not only led to wasted effort with duplicated code, but also exposed applications to potential security vulnerabilities.

**Introducing the Fabric Token SDK: Streamlining Tokenized Development (and Beyond)**

The Fabric Token SDK has evolved beyond its initial focus on Hyperledger Fabric. It now empowers developers with the following capabilities across various platforms, including permissioned blockchains like Fabric:

* **Tokenization Made Easy:** Create tokens representing any type of asset, be it physical or digital.
* **Privacy by Design:** Select the appropriate privacy level for your specific use case, without modifying your application logic.
* **Peer-to-Peer Transactions:** Orchestrate token transfers directly between users, streamlining the process.
* **Atomic Swaps:** Facilitate secure exchanges of different tokens without relying on intermediaries.
* **Transaction Auditing:** Review transactions before they are finalized, ensuring accuracy and compliance.
* **Interoperability:** Connect with token systems on other blockchain networks, fostering broader ecosystems.
* **Seamless Integration:** Add a token layer to existing applications, regardless of platform, with minimal effort.

With a robust Fabric Token SDK, developing secure and efficient enterprise-grade tokenized applications becomes a reality, offering flexibility for developers to choose the platform that best suits their needs.

# Development

For additional information about the development of the Token SDK, visit this [`section`](./docs/development/development.md).

# License

This project is licensed under the Apache 2 License - see the [`LICENSE`](LICENSE) file for details

[`fabric-token-sdk` Issues]: https://github.com/hyperledger-labs/fabric-token-sdk/issues
[GitHub discussions]: https://github.com/hyperledger-labs/fabric-token-sdk/discussions
