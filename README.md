# Fabric Token SDK
[![License](https://img.shields.io/badge/license-Apache%202-blue)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/hyperledger-labs/fabric-token-sdk)](https://goreportcard.com/badge/github.com/hyperledger-labs/fabric-token-sdk)
[![Go](https://github.com/hyperledger-labs/fabric-token-sdk/actions/workflows/tests.yml/badge.svg)](https://github.com/hyperledger-labs/fabric-token-sdk/actions/workflows/go.yml)
[![CodeQL](https://github.com/hyperledger-labs/fabric-token-sdk/actions/workflows/codeql-analysis.yml/badge.svg)](https://github.com/hyperledger-labs/fabric-token-sdk/actions/workflows/codeql-analysis.yml)

The `Fabric Token SDK` is a set of API and services that let developers create 
token-based distributed application on Hyperledger Fabric.

# Disclaimer

`Fabric Token SDK` has not been audited and is provided as-is, use at your own risk.
The project will be subject to rapid changes to complete the open-sourcing process, and  the list of features.

# Useful Links

- [`Documentation`](./docs/design.md): Discover the design principles of the Fabric Token SDK.
- [`Samples`](./samples/README.md): A collection of sample applications that demonstrate the use of the Fabric Token SDK.
- `Feedback`: Your help is the key to the success of the Fabric Token SDK. 
  - Submit your issues [`here`][`fabric-token-sdk` Issues]. 
  - Found a bug? Need help to fix an issue? You have a great idea for a new feature? Talk to us! You can reach us on
    [Discord](https://discord.gg/hyperledger) in #fabric-token-sdk.
  
- [`Fabric Smart Client`](https://github.com/hyperledger-labs/fabric-smart-client): The Token SDK leverages the 
  `Fabric Smart Client` for transaction orchestration, storing tokens and wallets, and more. Check it out.
- [Fabric Samples](https://github.com/hyperledger/fabric-samples/tree/main/token-sdk) Token SDK sample application is the
  quickest way to get a full network running with a REST API to issue, transfer and redeem tokens right away.

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
  
# Utilities

- [`tokengen`](./cmd/tokengen/README.md) is an utility for generating Fabric Token-SDK material.
  It is provided as a means of preconfiguring public parameters, token chaincode, and so.
  It would normally not be used in the operation of a production network.

# Additional Resources

- (March 17, 2022) [`Hyperledger in-Depth: Tokens in Hyperledger Fabric: What’s possible today and what’s coming`](https://www.hyperledger.org/learn/webinars/hyperledger-in-depth-tokens-in-hyperledger-fabric-whats-possible-today-and-whats-coming):
  Tokenizing the physical world is a hot blockchain topic in the industry, especially as it relates to the 
  trade of tokens as a basis of new forms of commerce. In this Hyperledger Foundation member webinar, 
  the IBM Research team describes in this webinar what tokenization use cases are possible with Hyperledger Fabric today, 
  and what enhancements are in the works (aka Fabric Token SDK).
- (October 12, 2023) [How to create a currency management app and deploy it to a Hyperledger Fabric network](https://www.youtube.com/watch?v=PX9SDva97vQ):
  In this comprehensive guide, we'll walk you through two essential aspects of the Fabric Token-SDK. Firstly, you'll learn how to develop a straightforward token application to manage a currency. You'll grasp the fundamentals of creating tokens, and implementing transaction logic using the Fabric Token-SDK. Once you've mastered the application development, we'll then show you how to effortlessly deploy it in your existing Fabric network, ensuring a seamless integration with your blockchain infrastructure. By the end of this tutorial, you'll be equipped with the skills to expand your blockchain capabilities and unleash the true potential of decentralized currency management. (Refers to [Fabric Samples](https://github.com/hyperledger/fabric-samples/tree/main/token-sdk))


<!-- markdown-link-check-disable -->
# Motivation

[Hyperledger Fabric]('https://wiki.hyperledger.org/display/fabric') is a permissioned, modular, and extensible open-source DLT platform. Fabric architecture follows a novel `execute-order-validate` paradigm that supports distributed execution of untrusted code in an untrusted environment. Indeed, Fabric-based distributed applications can be written in any general-purpose programming language.  
Fabric does not depend on a native cryptocurrency as it happens for existing blockchain platforms that require “smart-contracts” to be written in domain-specific languages or rely on a cryptocurrency.

Blockchain technologies are accelerating the shifting towards a decentralised economy. Cryptocurrencies are reshaping the financial landscape to the extent that even central banks are now testing the technology to propose what is known as the `central bank digital currency`. But it is more than this. Real-world assets are being tokenised as fungible or non-fungible assets represented by tokens on a blockchain. Thus enabling business opportunities to extract more value.

Developing token-based applications for Hyperledger Fabric is not easy. Fabric does not provide an out-of-the-box SDK that let developers create tokens that represents any kind of asset. Developers are left on their own and this exposes them to useless duplication of code and security vulnerabilities.

What would happen if the developers could use a `Fabric Token SDK` that let:
- Create tokens that represents any kind of asset (baked by a real-world asset or virtual);
- Choose the privacy level that best fits the use-case without changing the application logic;
- Orchestrate token transaction in a peer-to-peer fashion;
- Perform atomic swaps;
- Audit transactions before they get committed;
- Interoperate with token systems in other blockchain networks;
- Add a token layer to existing Fabric distributed application?

Developing Enterprise Token-based distributed applications would become simpler and more secure.
<!-- markdown-link-check-disable -->

# Testing Philosophy

[Write tests. Not too many. Mostly Integration](https://kentcdodds.com/blog/write-tests)

We also believe that when developing new functions running tests is preferable than running the application to verify the code is working as expected.

# Versioning

We use [`SemVer`](https://semver.org/) for versioning. For the versions available, see the [`tags on this repository`](https://github.com/hyperledger-labs/fabric-token-sdk/tags).

# License

This project is licensed under the Apache 2 License - see the [`LICENSE`](LICENSE) file for details

[`fabric-token-sdk` Issues]: https://github.com/hyperledger-labs/fabric-token-sdk/issues
[GitHub discussions]: https://github.com/hyperledger-labs/fabric-token-sdk/discussions
