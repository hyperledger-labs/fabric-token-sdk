# Gemini CLI Context: Fabric Token SDK

This directory contains the **Fabric Token SDK**, a project under Hyperledger Labs that provides APIs and services for building token-based distributed applications on Hyperledger Fabric and other platforms.

## Project Overview

*   **Purpose:** Simplify the development of tokenized applications with support for fungible/non-fungible tokens, privacy-preserving transactions (via Idemix/zkatdlog), and atomic swaps.
*   **Architecture:** Leverages the **Fabric Smart Client (FSC)** for transaction orchestration, storage, and wallet management.
    *   **Driver Consistency:** Core drivers (`fabtoken`, `zkatdlog`) follow a consistent architectural pattern using common interfaces defined in `token/driver` and shared logic in `token/core/common`.
*   **Core Components:**
    *   `token/`: The main SDK code.
        *   `core/`: Contains the specific driver implementations (`fabtoken`, `zkatdlog`) and shared logic.
        *   `driver/`: Defines the interfaces that drivers must implement.
        *   `services/`: High-level services (Identity, Network, Storage, TTX).
        *   `sdk/`: The public-facing API for developers building on top of the SDK.
    *   `integration/`: Integration tests and the Network Orchestrator (NWO).
*   **Key Technologies:** Go (1.24+), Hyperledger Fabric, Fabric Smart Client, Idemix, Mathlib, Ginkgo (testing), Cobra (CLI).

## Architecture & Design Patterns
*   **Driver Pattern:** The SDK uses a plugin architecture. `token/driver` defines the interfaces (Ports). `token/core` contains the Adapters (Implementations).
    *   **Goal:** Allow swapping the underlying token technology (e.g., from cleartext UTXO to ZK-UTXO) without changing the application logic.
    *   **Agent Tip:** When analyzing bugs, check if the issue is in the *interface definition* (`driver`) or a specific *implementation* (`core/fabtoken` vs `core/zkatdlog`).
*   **Service Pattern:** High-level logic is encapsulated in services (`token/services`).
    *   **Token Transaction (TTX):** The most critical service. It orchestrates the lifecycle of a token transaction (Request -> Assemble -> Sign -> Commit).
*   **Network Orchestrator (NWO):** Used for integration tests to programmatically define and spin up Fabric networks. It replaces manual `docker-compose` setups for testing.

## Building and Running

### Development Environment Setup
1.  **Install Tools:**
    ```bash
    make install-tools
    ```
    *Tools dependency source of truth is `tools/tools.go`.*
2.  **Download Fabric Binaries:**
    Critical for integration tests.
    ```bash
    make download-fabric
    export FAB_BINS=$PWD/../fabric/bin
    ```
    *Note: Do not store binaries inside the repo to avoid path issues.*
3.  **Prepare Docker Images:**
    Required for integration tests.
    ```bash
    make docker-images
    make testing-docker-images
    ```

### Common Commands
*   **Linting:**
    *   Check: `make lint`
    *   **Auto-fix:** `make lint-auto-fix` (Highly recommended before committing)
*   **Unit Tests:**
    *   Standard: `make unit-tests`
    *   Race Detection: `make unit-tests-race`
    *   Regression: `make unit-tests-regression`
*   **Integration Tests:**
    *   Format: `make integration-tests-<target>`
    *   Common Targets:
        *   `dlog-fabric-t1` (Zero-Knowledge, Basic)
        *   `fabtoken-fabric-t1` (Cleartext, Basic)
        *   `nft-dlog` (NFTs with Privacy)
        *   `dvp-fabtoken` (Delivery vs Payment)
    *   *Requires `FAB_BINS` to be set and Docker to be running.*
*   **Cleanup:**
    *   Artifacts: `make clean`
    *   Containers: `make clean-all-containers`
*   **Generate Mocks:** `go generate ./...` (uses `counterfeiter`)
*   **Tidy Modules:** `make tidy`

## Development Conventions

### Source Control & Contributions
*   **DCO Sign-off:** All commits **MUST** be signed off (`git commit -s`).
*   **Linear History:** Use a rebase workflow; avoid merge commits.
*   **License:** Apache License, Version 2.0.

### Coding Standards (Idiomatic Go)
*   **Error Handling:**
    *   Handle errors explicitly.
    *   Avoid `_` for error returns.
    *   Use `errors.Is` and `errors.As` for checking error types.
*   **Interfaces:**
    *   Define small, focused interfaces on the *consumer* side.
    *   Favor composition over inheritance.
*   **Concurrency:**
    *   Use goroutines and channels; avoid shared state where possible.
    *   Use `make unit-tests-race` to catch race conditions.
*   **Global Variables:** Avoid them to ensure testability and reduce side effects.
*   **Linting:** Zero-tolerance policy. Use `golangci-lint` (via `make lint`) to enforce standards.

### Documentation
*   **GoDocs:** All exported functions, structs, and interfaces **MUST** have clear, concise comments explaining their purpose, parameters, and return values.
*   **Test Documentation:** Test functions should briefly describe the scenario being tested (e.g., "Given X, when Y, then Z").
*   **System Documentation:** Any changes to the SDK's behavior, architecture, or public API **MUST** be reflected in the `docs/` directory. Keep diagrams (PUML) and markdown files in sync with the code.

### Testing Strategy
*   **Unit Tests:** Should be co-located with the code (`*_test.go`).
*   **Integration Tests:** Located in `integration/`. Use the **Network Orchestrator (NWO)** in `integration/nwo` to spin up ephemeral Fabric networks.
    *   **Fabric-X:** Tests starting with `fabricx` require additional setup (`make fxconfig configtxgen fabricx-docker-images`).
*   **Mocking:**
    *   Use `counterfeiter` for generating mocks.
    *   **Metrics:** Use `disabled.Provider` to avoid nil panics.
    *   **Tracing:** Use `noop.NewTracerProvider()`.

### Testing Best Practices
*   **Frameworks:** Use `github.com/stretchr/testify/assert` for values and `github.com/stretchr/testify/require` for error checking/termination.
*   **Table-Driven Tests:** Preferred for service logic to cover multiple edge cases efficiently.
*   **Mock Management:**
    *   Create a **Context Struct** (e.g., `TestContext`) to hold the object under test and all its mocks.
    *   Use a **Setup Helper** (e.g., `newTestContext(t)`) to initialize mocks with default "happy path" behaviors.
    *   This pattern (seen in `token/services/ttx`) drastically reduces boilerplate.
*   **Subtests:** Use `t.Run("Scenario Name", ...)` to group related assertions.
*   **Dependency Injection:** Design constructors to accept interfaces, facilitating easy mock injection.

## Key Files & Directories
*   `Makefile`: The central control hub. Read this to discover new targets.
*   `go.mod`: Project dependencies.
*   `tools/tools.go`: Tool dependencies (install with `make install-tools`).
*   `token/`: Core SDK logic.
    *   `driver/`: **Interfaces** defining the contract for token drivers.
    *   `core/`: **Implementations** of drivers.
        *   `fabtoken`: UTXO-based driver (cleartext).
        *   `zkatdlog`: UTXO-based driver with Zero-Knowledge Privacy (Idemix).
    *   `services/`: High-level services consumed by the SDK.
        *   `identity/`: Manages user identities and MSP interactions.
        *   `network/`: Handles communication with the Fabric network (or other DLTs).
        *   `ttx/`: **Token Transaction (TTX)** service for orchestration and atomic swaps.
    *   `sdk/`: Public API entry points for applications.
*   `integration/`: Integration tests.
    *   `nwo/`: **Network Orchestrator** for spinning up test networks.
    *   `token/`: Actual integration test suites (e.g., `fungible`, `nft`, `dvp`).

## Debugging & Advanced Testing
*   **Focused Integration Tests:**
    *   Use `TEST_FILTER` to run specific tests (uses Ginkgo labels).
        ```bash
        # Run only tests matching label "T1" in the dlog-fabric suite
        make integration-tests-dlog-fabric TEST_FILTER="T1"
        ```
    *   Alternatively, modify the test code: change `It("...", ...)` to `FIt("...", ...)` to focus, or `XIt("...", ...)` to skip. **Do not commit these changes.**
*   **Locating Logs:**
    *   **Integration Test Logs:** Found in the system's temporary directory (e.g., `/tmp/fsc-integration-<random>/...`).
    *   **Container Logs:** Use `docker logs <container_name>` to inspect running containers.
    *   **Tip:** To persist logs for debugging, you may temporarily modify the test to use `NewLocalTestSuite` (see `integration/token/test_utils.go`), which outputs to `./testdata`.
*   **Debugging Helpers:**
    *   **Wait for Input:** In integration tests, use `time.Sleep()` or a pause loop if you need to manually inspect the Docker state (containers will be torn down on test exit unless configured otherwise).
    *   **Preserve Network:** Check if the Make target supports a `no-cleanup` option or manually comment out the cleanup code in the test suite `AfterSuite`.

## Troubleshooting
*   **"Chaincode packaging failed":** Usually means `FAB_BINS` is not set or points to an invalid location.
*   **Docker errors:** Ensure `make testing-docker-images` has been run.
*   **Linting errors on commit:** Run `make lint-auto-fix`.
*   **Test timeouts:** Integration tests can be slow. Ensure you have allocated enough resources to Docker.

## Workflow Rules

- Before implementing any task, create a `plan.md` file in the project root containing:
    - A clear description of the goal
    - A numbered list of implementation steps
    - An "Implementation Progress" section with each step marked as `[ ] Pending`
- As you complete each step, update `plan.md` immediately, marking the step as `[x] Done` and adding a brief note about what was changed
- If you encounter a blocker or make a significant decision, log it under a `## Notes & Decisions` section in `plan.md`
- Mark the plan as `✅ COMPLETE` once all steps are done