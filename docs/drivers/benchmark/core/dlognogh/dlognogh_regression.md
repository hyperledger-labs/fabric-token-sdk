# ZK-ATDLOG Regression Tests

This document explains the regression testing for the Zero-Knowledge Anonymous Token Discrete Logarithm (ZK-ATDLOG) validator implementation.

> **Related Documentation:**  
> - [Testing Architecture](./dlognogh_architecture.md) - Understanding the test layers  
> - [Running Benchmarks](./dlognogh.md) - How to run performance benchmarks

## Overview

Regression tests ensure **backwards compatibility** of the ZK-ATDLOG validator by verifying that previously generated token requests remain valid across code changes. These tests use pre-recorded test vectors containing serialized token requests that must continue to validate correctly.

**Location:** `token/core/zkatdlog/nogh/v1/validator/regression/`

## Purpose

- **Backwards Compatibility**: Ensure new code changes don't break validation of existing token requests
- **Protocol Stability**: Verify that the cryptographic protocol remains consistent across versions
- **Change Detection**: Identify when modifications require regenerating test data

## Test Structure

### Test Data Organization

```
testdata/
├── zero/
│   ├── 32-BLS12_381_BBS_GURVY/     # 32-bit range proofs, BLS12_381 curve
│   │   ├── params.txt              # Base64-encoded public parameters
│   │   ├── transfers_i1_o1/        # Transfer: 1 input, 1 output
│   │   │   └── testdata.json       # Aggregated test cases (64 cases)
│   │   ├── transfers_i1_o2/        # Transfer: 1 input, 2 outputs
│   │   │   └── testdata.json
│   │   ├── transfers_i2_o1/        # Transfer: 2 inputs, 1 output
│   │   │   └── testdata.json
│   │   ├── transfers_i2_o2/        # Transfer: 2 inputs, 2 outputs
│   │   │   └── testdata.json
│   │   ├── issues_i1_o1/           # Issue operations
│   │   │   └── testdata.json
│   │   ├── redeems_i1_o1/          # Redeem operations
│   │   │   └── testdata.json
│   │   └── swaps_i1_o1/            # Swap operations
│   │       └── testdata.json
│   ├── 32-BN254/                   # 32-bit range proofs, BN254 curve
│   ├── 64-BLS12_381_BBS_GURVY/     # 64-bit range proofs, BLS12_381 curve
│   └── 64-BN254/                   # 64-bit range proofs, BN254 curve
```

### Test Vector Format

Each `testdata.json` file contains an aggregated map of 64 test cases:
```json
{
  "0": {
    "req_raw": "<base64-encoded token request>",
    "txid": "<transaction ID>",
    "metadata": "<base64-encoded metadata>",
    "inputs": [[<serialized-token-bytes>], [...]]
  },
  "1": {
    "req_raw": "...",
    "txid": "...",
    "metadata": "...",
    "inputs": [[...], [...]]
  },
  ...
  "63": { ... }
}
```

**Fields:**
- `req_raw`: Base64-encoded serialized token request
- `txid`: Transaction ID for the request
- `metadata`: Base64-encoded token request metadata (for auditor validation)
- `inputs`: Nested array of serialized input tokens (for auditor validation)

## Running Regression Tests

### Run All Tests

```bash
cd token/core/zkatdlog/nogh/v1/validator/regression
go test -v
```

### Run Specific Configuration

```bash
# Run only 32-bit BLS12_381 tests
go test -v -run "TestRegression/testdata/32-BLS12_381_BBS_GURVY"

# Run only transfer tests
go test -v -run "TestRegression.*transfers"

# Run specific input/output combination
go test -v -run "TestRegression.*transfers_i2_o2"
```

### Parallel Execution

The tests run in parallel by default. To control parallelism:

```bash
# Run with specific number of parallel tests
go test -v -parallel 4
```

## Test Coverage

The regression suite tests:

- **4 Action Types**: transfers, issues, redeems, swaps
- **4 Input/Output Combinations**: i1_o1, i1_o2, i2_o1, i2_o2
- **4 Configurations**: 2 bit sizes (32, 64) × 2 curves (BLS12_381, BN254)
- **64 Test Cases per Configuration**: Each `testdata.json` contains 64 test cases
- **Total Test Vectors**: 4,096 test cases across all configurations

## Generating New Test Data

When code changes require regenerating test vectors:

### 1. Use the Generator

```bash
cd token/core/zkatdlog/nogh/v1/regression/testdata/zero/generator
go generate
```

This will generate `testdata.json` files in each configuration directory, with each file containing all 64 test cases in an aggregated format.

### 2. Document the Change

Update `changes.md` with:
- Commit hash where change occurred
- Description of what changed
- Reason for regeneration

Example:
```markdown
## With respect to commit `<commit-hash>`

Description of the change that required test data regeneration.
```

### 3. Commit New Test Data

```bash
git add token/core/zkatdlog/nogh/v1/regression/testdata/
git commit -m "Regenerate regression test data: <reason>"
```

## When to Regenerate Test Data

Regenerate test vectors when:

- **Serialization format changes**: Any modification to how token requests are serialized
- **Cryptographic changes**: Updates to proof generation or verification algorithms
- **Protocol updates**: Changes to the token protocol itself
- **Bug fixes**: Corrections that affect the output format

## Related Tests

- **Layer 3 Service Layer - Transfer Validator Service Benchmarks**: Performance testing of the same validation logic
  - `BenchmarkValidatorTransfer`
  - `TestParallelBenchmarkValidatorTransfer`
- See [dlognogh_architecture.md](./dlognogh_architecture.md) for the complete testing architecture