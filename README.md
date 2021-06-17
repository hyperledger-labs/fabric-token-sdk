# Fabric Token SDK
[![License](https://img.shields.io/badge/license-Apache%202-blue)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/hyperledger-labs/fabric-token-sdk)](https://goreportcard.com/badge/github.com/hyperledger-labs/fabric-token-sdk)
[![Go](https://github.com/hyperledger-labs/fabric-token-sdk/actions/workflows/go.yml/badge.svg)](https://github.com/hyperledger-labs/fabric-token-sdk/actions/workflows/go.yml)
[![CodeQL](https://github.com/hyperledger-labs/fabric-token-sdk/actions/workflows/codeql-analysis.yml/badge.svg)](https://github.com/hyperledger-labs/fabric-token-sdk/actions/workflows/codeql-analysis.yml)

The `Fabric Token SDK` is a set of API and services that let developers create 
token-based distributed application on Hyperledger Fabric.

# Useful Links

- [`Documentation`](./docs/design.md): Discover the design principles of the Fabric Token SDK.
- [`Examples`](./integration/README.md): Learn how to use the Fabric Smart Client via examples. There is nothing better than this.
- `Feedback`: Your help is the key to the success of the Fabric Token SDK. 
  - Submit your issues [`here`][`fabric-token-sdk` Issues]. 
  - If you have any questions, queries or comments, find us on [GitHub discussions].
  
- [`Fabric Smart Client`](https://github.com/hyperledger-labs/fabric-smart-client): The Token SDK leverages the 
  `Fabric Smart Client` for transaction orchestration, storing tokens and wallets, and more. Check it out. 
  
# Disclaimer

`Fabric Token SDK` has not been audited and is provided as-is, use at your own risk. 
The project will be subject to rapid changes to complete the open-sourcing process, and
the list of features.

# Motivation

[Hyperledger Fabric]('https://www.hyperledger.org/use/fabric') is a permissioned, modular, and extensible open-source DLT platform. Fabric architecture follows a novel `execute-order-validate` paradigm that supports distributed execution of untrusted code in an untrusted environment. Indeed, Fabric-based distributed applications can be written in any general-purpose programming language.  
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

# Use the Fabric Token SDK

## Install

The `Fabric Token SDK` can be downloaded using `go get` as follows:
 ```
go get github.com/hyperledger-labs/fabric-token-sdk
```

The above command clones the repo under `$GOPATH/github.com/hyperledger-labs/fabric-token-sdk`.

We recommend to use `go 1.14.13`. We are testing the Token SDK also against more recent versions of the go-sdk to make sure the Token SDK works properly.

## Makefile

The Token SDK is equipped with a `Makefile` to simplify some tasks.
Here is the list of commands available.

- `make checks`: check code formatting, style, and licence header.
- `make unit-tests`: execute the unit-tests.
- `make integration-tests`: execute the integration tests. The integration tests use `ginkgo`. Please, make sure that `$GOPATH/bin` is in your `PATH` env variable.
- `make clean`: clean the docker environment, useful for testing.

Executes the above from `$GOPATH/github.com/hyperledger-labs/fabric-token-sdk`.

## Testing Philosophy

[Write tests. Not too many. Mostly Integration](https://kentcdodds.com/blog/write-tests)

We also believe that when developing new functions running tests is preferable than running the application to verify the code is working as expected.

## Versioning

We use [`SemVer`](https://semver.org/) for versioning. For the versions available, see the [`tags on this repository`](https://github.com/hyperledger-labs/fabric-token-sdk/tags).

# License

This project is licensed under the Apache 2 License - see the [`LICENSE`](LICENSE) file for details

[`fabric-token-sdk` Issues]: https://github.com/hyperledger-labs/fabric-token-sdk/issues
[GitHub discussions]: https://github.com/hyperledger-labs/fabric-token-sdk/discussions
