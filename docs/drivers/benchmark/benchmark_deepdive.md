# ZK-ATDLOG Testing Architecture

This document explains the layered testing architecture for the Zero-Knowledge Anonymous Token Discrete Logarithm (ZK-ATDLOG) implementation in the Fabric Token SDK.

## Overview

The ZK-ATDLOG tests are organized in a **layered "onion" architecture**, where each layer tests a different level of abstraction - from low-level cryptographic operations to high-level service APIs. This design allows developers to:

- **Pinpoint performance bottlenecks** at specific abstraction levels
- **Isolate bugs** to particular components
- **Ensure backwards compatibility** across protocol versions
- **Measure end-to-end performance** in realistic scenarios

## Architecture Layers

```
┌─────────────────────────────────────────────────────────────────┐
│ Layer 4: Service Layer (Outermost)                             │
│ Location: token/core/zkatdlog/nogh/v1/                         │
│ Tests: BenchmarkTransferServiceTransfer                         │
│        TestParallelBenchmarkTransferServiceTransfer             │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ Layer 3: Validator Layer                                        │
│ Location: token/core/zkatdlog/nogh/v1/validator/               │
│ Tests: TestParallelBenchmarkValidatorTransfer (performance)     │
│        regression_test.go (backwards compatibility)             │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ Layer 2: Action Operations (Transfer & Issue)                  │
│ Locations: token/core/zkatdlog/nogh/v1/transfer/               │
│            token/core/zkatdlog/nogh/v1/issue/                   │
│ Tests: BenchmarkSender (transfer generation)                    │
│        BenchmarkVerificationSenderProof (transfer verification) │
│        BenchmarkIssuer (issue generation)                       │
│        BenchmarkProofVerificationIssuer (issue verification)    │
│        TestParallelBenchmark* variants                          │
└─────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────┐
│ Layer 1: Core Cryptography (Innermost)                         │
│ Location: token/core/zkatdlog/nogh/v1/transfer/                │
│ Tests: BenchmarkTransferProofGeneration                         │
│        TestParallelBenchmarkTransferProofGeneration             │
└─────────────────────────────────────────────────────────────────┘
```

---

## Layer 1: Core Cryptographic Operations

**Location:** `token/core/zkatdlog/nogh/v1/transfer/`

### Tests
- `BenchmarkTransferProofGeneration`
- `TestParallelBenchmarkTransferProofGeneration`

### What They Measure
Pure zero-knowledge proof generation and serialization, isolated from all other operations.

### Includes
- ZK proof computation (range proofs, equality proofs, etc.)
- Proof serialization to bytes

### Excludes
- Token creation
- Action assembly
- Metadata handling
- Validation logic

### Use Cases
- **Optimize cryptographic algorithms**: Identify bottlenecks in proof generation
- **Compare curve performance**: Measure BN254 vs BLS12-381
- **Evaluate bit-size impact**: Compare 32-bit vs 64-bit range proofs
- **Assess cryptographic library changes**: Detect performance regressions in mathlib

### Example Command
```bash
cd token/core/zkatdlog/nogh/v1/transfer
go test -bench=BenchmarkTransferProofGeneration -benchtime=10s
```

---

## Layer 2: Transfer Action Generation & Verification

**Location:** `token/core/zkatdlog/nogh/v1/transfer/`

### Generation Tests
- `BenchmarkSender`
- `BenchmarkParallelSender`
- `TestParallelBenchmarkSender`

### Verification Tests
- `BenchmarkVerificationSenderProof`
- `BenchmarkVerificationParallelSenderProof`
- `TestParallelBenchmarkVerificationSenderProof`

### What They Measure

#### Generation Side (Sender)
Complete transfer action creation from inputs to serialized output.

**Includes:**
- ZK proof generation (Layer 1)
- Input token handling
- Output token creation
- Action serialization

**Excludes:**
- Verification
- Validation
- Service orchestration

#### Verification Side (Verifier)
Deserialization and cryptographic verification of transfer actions.

**Includes:**
- Action deserialization
- ZK proof verification

**Excludes:**
- Token validation
- Signature checks
- Business logic

### Use Cases
- **Optimize sender operations**: Measure time to create a transfer
- **Optimize verifier operations**: Measure time to verify a proof
- **Evaluate serialization overhead**: Compare proof generation vs total action creation
- **Test concurrent proof generation**: Measure multi-threaded performance

### Example Commands
```bash
# Sender benchmarks
go test -bench=BenchmarkSender -benchtime=10s

# Verifier benchmarks
go test -bench=BenchmarkVerificationSenderProof -benchtime=10s

# Parallel execution test
go test -run=TestParallelBenchmarkSender -v
```

---

## Layer 2: Action Operations (Transfer & Issue)

**Locations:**
- `token/core/zkatdlog/nogh/v1/transfer/` (Transfer operations)
- `token/core/zkatdlog/nogh/v1/issue/` (Issue operations)

### Transfer Tests
- `BenchmarkSender`
- `BenchmarkParallelSender`
- `TestParallelBenchmarkSender`
- `BenchmarkVerificationSenderProof`
- `BenchmarkVerificationParallelSenderProof`
- `TestParallelBenchmarkVerificationSenderProof`

### Issue Tests
- `BenchmarkIssuer`
- `BenchmarkProofVerificationIssuer`

### What They Measure

Both Transfer and Issue operations are at the same abstraction level - they are different **action types** that operate on tokens.

#### Transfer Operations
Complete transfer action creation and verification for moving tokens between parties.

**Generation Side (Sender):**
- ZK proof generation (Layer 1)
- Input token handling
- Output token creation
- Action serialization

**Verification Side (Verifier):**
- Action deserialization
- ZK proof verification

#### Issue Operations
Token issuance operations for creating new tokens.

**Includes:**
- Issue action generation
- Issue proof verification
- Token creation with commitments
- Blind signature operations (for privacy)

### Use Cases

#### Transfer
- **Optimize sender operations**: Measure time to create a transfer
- **Optimize verifier operations**: Measure time to verify a proof
- **Evaluate serialization overhead**: Compare proof generation vs total action creation
- **Test concurrent proof generation**: Measure multi-threaded performance

#### Issue
- **Optimize issuance performance**: Measure time to issue new tokens
- **Compare issue vs transfer**: Understand performance differences between action types
- **Evaluate issuer-specific cryptography**: Measure blind signature operations

### Example Commands

#### Transfer Benchmarks
```bash
cd token/core/zkatdlog/nogh/v1/transfer

# Sender benchmarks
go test -bench=BenchmarkSender -benchtime=10s

# Verifier benchmarks
go test -bench=BenchmarkVerificationSenderProof -benchtime=10s

# Parallel execution test
go test -run=TestParallelBenchmarkSender -v
```

#### Issue Benchmarks
```bash
cd token/core/zkatdlog/nogh/v1/issue
go test -bench=BenchmarkIssuer -benchtime=10s
go test -bench=BenchmarkProofVerificationIssuer -benchtime=10s
```

---

## Layer 3: Validator Layer

**Location:** `token/core/zkatdlog/nogh/v1/validator/`

### Performance Tests
- `BenchmarkValidatorTransfer`
- `TestParallelBenchmarkValidatorTransfer`

### Compatibility Tests
- `regression_test.go` (in `validator/regression/`)

### What They Measure

#### Performance Tests
Complete validation pipeline performance, including all cryptographic and business logic checks.

**Includes:**
- Action deserialization (Layer 2)
- ZK proof verification (Layer 1)
- Signature verification
- Token validation
- Business logic checks (double-spend prevention, balance checks, etc.)

**Excludes:**
- High-level service orchestration
- Wallet management
- Identity resolution

#### Regression Tests
Backwards compatibility verification using pre-recorded "golden" test vectors.

**Includes:**
- Same validation pipeline as performance tests
- 1,024 pre-recorded requests (64 vectors × 16 configurations)
- Multiple curves (BN254, BLS12-381)
- Multiple bit sizes (32-bit, 64-bit)
- Multiple action types (transfers, issues, redeems, swaps)

### Use Cases

#### Performance Tests
- **Measure complete validation time**: End-to-end validator performance
- **Identify validation bottlenecks**: Profile which validation step is slowest
- **Test concurrent validation**: Measure multi-threaded validator performance

#### Regression Tests
- **Ensure backwards compatibility**: Verify old requests still validate
- **Detect breaking changes**: Catch protocol modifications
- **Validate serialization format**: Ensure format stability across versions
- **Test cryptographic library updates**: Verify mathlib upgrades don't break compatibility

### Example Commands
```bash
# Performance benchmarks
cd token/core/zkatdlog/nogh/v1/validator
go test -bench=BenchmarkValidatorTransfer -benchtime=10s

# Regression tests
cd token/core/zkatdlog/nogh/v1/validator/regression
go test -v

# Run specific regression configuration
go test -run="TestRegression/testdata/32-BLS12_381_BBS_GURVY-transfers_i2_o2" -v
```

### Regression Test Data Structure
```
validator/regression/testdata/
├── 32-BLS12_381_BBS_GURVY/
│   ├── params.txt                    # Base64-encoded public parameters
│   ├── transfers_i1_o1/              # 1 input, 1 output
│   │   ├── output.0.json
│   │   ├── output.1.json
│   │   └── ... (64 files)
│   ├── transfers_i1_o2/
│   ├── transfers_i2_o1/
│   ├── transfers_i2_o2/
│   ├── issues_i1_o1/
│   ├── redeems_i1_o1/
│   └── swaps_i1_o1/
├── 64-BLS12_381_BBS_GURVY/
├── 32-BN254/
└── 64-BN254/
```

Each `output.<n>.json` contains:
```json
{
  "req_raw": "<base64-encoded-token-request>",
  "txid": "<transaction-id>"
}
```

---

## Layer 4: Service Layer

**Location:** `token/core/zkatdlog/nogh/v1/`

### Tests
- `BenchmarkTransferServiceTransfer`
- `TestParallelBenchmarkTransferServiceTransfer`

### What They Measure
Complete end-to-end transfer operation through the high-level Transfer Service API.

### Includes
- All lower layers (1-3)
- Token selection from wallet
- Wallet management
- Identity handling
- Service orchestration
- Transaction assembly

### Excludes
- Network communication
- Persistence
- Finality handling

### Use Cases
- **Measure real-world performance**: End-to-end application-level timing
- **Evaluate service overhead**: Compare to Layer 3 to measure orchestration cost
- **Test realistic scenarios**: Include wallet and identity operations
- **Benchmark application-level APIs**: Measure what developers actually call

### Example Commands
```bash
cd token/core/zkatdlog/nogh/v1
go test -bench=BenchmarkTransferServiceTransfer -benchtime=10s
go test -run=TestParallelBenchmarkTransferServiceTransfer -v
```

---

## Test Execution Patterns

### Sequential vs Parallel

#### Sequential Benchmarks (`Benchmark*`)
```go
func BenchmarkSender(b *testing.B) {
    // Runs b.N iterations sequentially
    // Measures single-threaded performance
}
```

**Use for:**
- Baseline performance measurement
- Comparing algorithm changes
- Profiling with `go test -cpuprofile`

#### Parallel Benchmarks (`BenchmarkParallel*`)
```go
func BenchmarkParallelSender(b *testing.B) {
    b.RunParallel(func(pb *testing.PB) {
        // Runs on multiple goroutines
        // Measures concurrent performance
    })
}
```

**Use for:**
- Multi-core scalability testing
- Contention detection
- Real-world concurrent load simulation

#### Custom Parallel Tests (`TestParallelBenchmark*`)
```go
func TestParallelBenchmarkSender(t *testing.T) {
    // Custom parallel execution with configurable goroutines
    // Measures multi-goroutine contention
}
```

**Use for:**
- Controlled concurrency testing
- Race condition detection
- Stress testing with specific goroutine counts

---

## Performance Comparison Table

| Layer | Test | What It Measures | Typical Time (2 inputs, 2 outputs, BLS12-381, 64-bit) |
|-------|------|------------------|-------------------------------------------------------|
| 1 | `BenchmarkTransferProofGeneration` | Pure ZK proof | ~50-100ms |
| 2 | `BenchmarkSender` | Proof + action packaging (transfer) | ~60-120ms |
| 2 | `BenchmarkIssuer` | Issue action generation | ~60-120ms |
| 2 | `BenchmarkVerificationSenderProof` | Proof verification only | ~30-60ms |
| 3 | `BenchmarkValidatorTransfer` | Full validation | ~40-80ms |
| 4 | `BenchmarkTransferServiceTransfer` | Complete service API | ~80-150ms |

*Note: Times are approximate and vary based on hardware, curve, bit-size, and number of inputs/outputs.*

---

## Running Benchmarks

### Basic Benchmark
```bash
cd token/core/zkatdlog/nogh/v1/transfer
go test -bench=. -benchtime=10s
```

### With Memory Profiling
```bash
go test -bench=BenchmarkSender -benchmem -memprofile=mem.out
go tool pprof mem.out
```

### With CPU Profiling
```bash
go test -bench=BenchmarkSender -cpuprofile=cpu.out
go tool pprof cpu.out
```

### Specific Configuration
```bash
# Run only 32-bit BLS12-381 benchmarks
go test -bench="32.*BLS12_381" -benchtime=5s
```

### Parallel Execution
```bash
# Run parallel benchmark with specific GOMAXPROCS
GOMAXPROCS=8 go test -bench=BenchmarkParallelSender
```

---

## Regression Test Management

### Running Regression Tests
```bash
cd token/core/zkatdlog/nogh/v1/validator/regression
go test -v
```

### Generating New Test Vectors
If you need to regenerate test vectors (e.g., after a protocol upgrade):

1. Navigate to the generator:
   ```bash
   cd token/core/zkatdlog/nogh/v1/validator/regression/testdata/generator
   ```

2. Run the generator (implementation-specific)

3. Commit the new artifacts:
   ```bash
   git add token/core/zkatdlog/nogh/v1/validator/regression/testdata/
   git commit -m "Update regression test vectors for protocol v2"
   ```

### Understanding Regression Test Failures

If regression tests fail, it means:
- ✗ **Backwards compatibility is broken**
- ✗ Previously valid requests no longer validate
- ✗ Serialization format has changed
- ✗ Cryptographic protocol has been modified

**Action required:**
1. Determine if the change is intentional (protocol upgrade)
2. If intentional: Regenerate test vectors and document the breaking change
3. If unintentional: Fix the bug that broke compatibility

---

## Best Practices

### When to Use Each Layer

| Scenario | Recommended Layer | Reason |
|----------|------------------|---------|
| Optimizing proof generation | Layer 1 | Isolates cryptographic operations |
| Debugging serialization issues | Layer 2 | Tests action packaging |
| Comparing transfer vs issue | Layer 2 | Both are action types |
| Measuring validation performance | Layer 3 | Complete validation pipeline |
| Testing backwards compatibility | Layer 3 (regression) | Ensures protocol stability |
| Benchmarking application performance | Layer 4 | Realistic end-to-end timing |

### Development Workflow

1. **Start with Layer 1**: Optimize core cryptography first
2. **Move to Layer 2**: Ensure efficient action generation/verification (transfer & issue)
3. **Validate at Layer 3**: Confirm complete validation works
4. **Test at Layer 4**: Measure real-world performance
5. **Run regression tests**: Ensure no breaking changes

### Continuous Integration

Recommended CI pipeline:
```yaml
- name: Unit Tests
  run: go test ./token/core/zkatdlog/nogh/v1/...

- name: Regression Tests
  run: go test ./token/core/zkatdlog/nogh/v1/validator/regression/...

- name: Benchmark Smoke Test
  run: go test -bench=. -benchtime=1s ./token/core/zkatdlog/nogh/v1/...
```

---

## Troubleshooting

### Benchmark Variability
If benchmarks show high variance:
- Run with longer `-benchtime` (e.g., `30s`)
- Disable CPU frequency scaling
- Close other applications
- Use `benchstat` to compare results statistically

### Regression Test Failures
If regression tests fail unexpectedly:
1. Check if mathlib was updated
2. Verify public parameters haven't changed
3. Review recent serialization changes
4. Compare with previous test vectors

### Performance Degradation
If benchmarks show performance regression:
1. Compare with baseline using `benchstat`
2. Profile with `-cpuprofile` to find bottlenecks
3. Check if the regression is in a specific layer
4. Review recent code changes in that layer

---

## Additional Resources

- **Benchmark Documentation**: See `token/services/benchmark/README.md`
- **ZK-ATDLOG Protocol**: See `docs/tokenapi.md`
- **Cryptographic Details**: See research papers in `docs/`
- **Profiling Guide**: See `docs/development/monitoring.md`

---

## Summary

The ZK-ATDLOG testing architecture provides:

✅ **Layered testing** from cryptography to services  
✅ **Performance benchmarking** at each abstraction level  
✅ **Backwards compatibility** verification via regression tests  
✅ **Parallel execution** testing for concurrency issues  
✅ **Comprehensive coverage** of all operation types  

This architecture enables developers to optimize, debug, and validate the ZK-ATDLOG implementation with precision and confidence.