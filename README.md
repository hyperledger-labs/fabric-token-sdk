# Fabric Token SDK
[![License](https://img.shields.io/badge/license-Apache%202-blue)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/hyperledger-labs/fabric-token-sdk)](https://goreportcard.com/badge/github.com/hyperledger-labs/fabric-token-sdk)
[![Go](https://github.com/hyperledger-labs/fabric-token-sdk/actions/workflows/go.yml/badge.svg)](https://github.com/hyperledger-labs/fabric-token-sdk/actions/workflows/go.yml)
[![CodeQL](https://github.com/hyperledger-labs/fabric-token-sdk/actions/workflows/codeql-analysis.yml/badge.svg)](https://github.com/hyperledger-labs/fabric-token-sdk/actions/workflows/codeql-analysis.yml)

The `Fabric Token SDK` is a set of API and services that let developers create 
token-based distributed application on Hyperledger Fabric.

# Useful Links

- [`Documentation`](./docs/design.md): Discover the design principles of the Fabric Token SDK.
- [`Samples`](./samples/README.md): A collection of sample applications that demonstrate the use of the Fabric Token SDK.
- `Feedback`: Your help is the key to the success of the Fabric Token SDK. 
  - Submit your issues [`here`][`fabric-token-sdk` Issues]. 
  - If you have any questions, queries or comments, find us on [GitHub discussions].
  
- [`Fabric Smart Client`](https://github.com/hyperledger-labs/fabric-smart-client): The Token SDK leverages the 
  `Fabric Smart Client` for transaction orchestration, storing tokens and wallets, and more. Check it out. 
  
# Disclaimer

`Fabric Token SDK` has not been audited and is provided as-is, use at your own risk. 
The project will be subject to rapid changes to complete the open-sourcing process, and
the list of features.

# Getting Help

Found a bug? Need help to fix an issue? You have a great idea for a new feature? Talk to us! You can reach us on
[Discord](https://discord.gg/hyperledger) in #fabric-token-sdk.

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

# Testing Philosophy

[Write tests. Not too many. Mostly Integration](https://kentcdodds.com/blog/write-tests)

We also believe that when developing new functions running tests is preferable than running the application to verify the code is working as expected.

# Versioning

We use [`SemVer`](https://semver.org/) for versioning. For the versions available, see the [`tags on this repository`](https://github.com/hyperledger-labs/fabric-token-sdk/tags).

# License

This project is licensed under the Apache 2 License - see the [`LICENSE`](LICENSE) file for details

[`fabric-token-sdk` Issues]: https://github.com/hyperledger-labs/fabric-token-sdk/issues
[GitHub discussions]: https://github.com/hyperledger-labs/fabric-token-sdk/discussions
