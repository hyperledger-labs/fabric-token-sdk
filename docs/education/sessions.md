## Fabric Token SDK - Session 1 - 29.10.2025
**Presenter:** Angelo de Caro  

### Token SDK Overview
The Token SDK is a software library designed to facilitate token-related transactions on the Hyperledger Fabric platform. It provides a set of APIs and services for building token-based applications on Fabric and potentially other platforms. Initially embedded within the Fabric peer, the SDK manages complex cryptographic operations such as zero-knowledge proof generation and wallet key management. Subsequently, the architecture was refactored to operate at the application layer, decoupling it from the core Fabric infrastructure. This design abstracts Fabric as a black box, delegating cryptographic computations and secret key handling externally, thereby enhancing modularity and developer flexibility.

### Token SDK Folders Overview

Referring to the project tree of [fabric-token-sdk](https://github.com/hyperledger-labs/fabric-token-sdk/).

#### .github

Contains GitHub-related configuration files, primarily for Continuous Integration (CI) workflows. This includes scripts for code analysis, code linting, documentation link checks, and running tests.

#### ci

Holds scripts specifically for CI tasks, especially for HSM (Hardware Security Module) integration. It includes setup and test scripts for SoftHSM, a software-based HSM simulator used to test secret key handling.

#### cmd

Includes command-line tools, notably Token Gen, which is used primarily in development and testing environments to generate cryptographic material needed by the Token SDK. This tool supports setting up token infrastructure but is not intended for production use.

#### docs

Contains documentation related to the Token SDK, explaining concepts, usage, and other relevant information.

#### integration

Contains end-to-end integration tests. These tests bootstrap minimal instances of Fabric along with required components to validate the full token transaction workflows.

#### token

This is the core folder containing the actual implementation of the Token SDK. Most of the business logic and functionalities used by projects like IDAP come from this directory.

#### tools

Contains auxiliary tools used within the project’s build and maintenance processes, such as linters, license checks, and other utility scripts.

#### Makefile

(Not a folder but a key file) Defines a set of commands for building, testing, cleaning, and maintaining the project. It helps automate tasks like dependency installation and housekeeping.

### Overview of Token SDK Components

The Token SDK is composed of several core components that interact to provide a modular and secure framework for token-based operations. The following gives a high-level view of the main architectural elements within the token folder.

![Token SDK Stack](https://raw.githubusercontent.com/hyperledger-labs/fabric-token-sdk/main/docs/imgs/stack.png)


#### Token API

The Token API serves as the main entry point for developers interacting with the Token SDK. It provides a set of abstract interfaces that allow developers to perform token-related operations such as issuing, transferring, or redeeming tokens without depending on a specific backend implementation. 

The Token API is designed to be abstract and backend-agnostic. It does not include any concepts specific to Hyperledger Fabric, centralized databases, zero-knowledge proofs, trusted execution environments, or other specialized technologies. Instead, the Token API exposes only generic concepts for token operations, allowing it to be flexible and adaptable to different backends or implementations.

This abstraction ensures flexibility and portability across different environments or driver implementations. Developers can invoke token functionalities without dealing directly with the underlying complexity of the fabric or cryptographic drivers.

You can find the Token API code in the files under the [fabric-token-sdk
/token/](https://github.com/hyperledger-labs/fabric-token-sdk/tree/main/token) folder. For example, see the [fabric-token-sdk/token/actions.go](https://github.com/hyperledger-labs/fabric-token-sdk/blob/main/token/actions.go) file.

In general, all Token API files consist of structs that wrap interfaces from the Driver API. For example see the [fabric-token-sdk/token/actions.go](https://github.com/hyperledger-labs/fabric-token-sdk/blob/main/token/actions.go) file, you will find:

```go
// IssueAction represents an action that issues tokens.
type IssueAction struct {
    a driver.IssueAction
}
```

#### Driver API

The Driver API provides concrete implementations of the token operations. Each driver defines how token actions (e.g., issue, transfer) are executed on a particular backend (refers to the underlying system or component that executes the actual token operations which in our case is Hyperledger Fabric).
The Token API acts as a mediator between the developer and the driver, enforcing clear boundaries to prevent misuse or unsafe interactions with low-level driver components.

This separation of concerns enables:

- Easier switching between different drivers or backends.

- Safer and more stable use of token functionalities.

- Better encapsulation and modularity in the SDK design.

You can find the code of Driver API in the [fabric-token-sdk/token
/driver](https://github.com/hyperledger-labs/fabric-token-sdk/tree/main/token/driver) folder.

#### Driver 
The Driver is the concrete implementation of the Driver API responsible for executing token operations on a specific backend, in this case, Hyperledger Fabric. It handles all the low-level interactions with Fabric, such as submitting transactions, managing ledger state, and performing cryptographic checks.

This layer encapsulates Fabric-specific details, allowing upper layers (like the Token API) to remain backend-agnostic and focused on business logic.

Key Characteristics:

- Implements the backend logic to carry out Fabric operations, including ledger updates, transaction endorsement, and commit phases.

- Hides complex Fabric internals from the rest of the system, providing a clean abstraction.

- Supports multiple driver implementations, enabling modularity and flexibility across different backends or Fabric configurations.

You can find the code of Driver in the [fabric-token-sdk/token/core](https://github.com/hyperledger-labs/fabric-token-sdk/tree/main/token/core) folder.


#### Services

While the Token API provides low-level access to token operations, it doesn’t automate or simplify complex workflows. This is where Services come in.

Services act as utility layers that help developers assemble multiple token operations into higher level, meaningful tasks such as building a complete token transaction involving several steps. They manage the correct sequencing and invocation of the Token API calls, reducing complexity and making the developer’s job easier.

In short, Services provide helpful abstractions and workflows on top of the Token API to streamline common use cases and improve developer productivity.

You can find the code for the various services in the [fabric-token-sdk/token
/services](https://github.com/hyperledger-labs/fabric-token-sdk/tree/main/token/services) folder. Inside this folder, there are two driver implementations: one for Fabtoken and the other for Zkatdlog.
