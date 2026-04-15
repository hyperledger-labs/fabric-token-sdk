# Transfer Service Benchmark

This document describes how to benchmark the Transfer Service at the node level using the Fabric Smart Client (FSC) framework.

## Overview

The Transfer Service benchmarks measure the performance of token transfer validation through different architectural layers:

1. **Local View Benchmark** - Direct in-process validation (no network overhead)
2. **API Benchmark** - Validation through FSC's View API (local node)
3. **gRPC API Benchmark** - Validation through gRPC client-server architecture (network overhead included)
4. **Distributed Benchmark** - Two-node setup with separate client and server machines

These benchmarks complement the [core driver benchmarks](core/dlognogh/dlognogh.md) by testing the complete service layer including FSC integration, network communication, and node orchestration.

## Motivation

The Transfer Service benchmarks serve several critical purposes:

### Performance Validation
- **End-to-End Metrics**: Measure complete transaction validation pipeline including deserialization, cryptographic verification, signature checks, and business logic
- **Network Overhead**: Quantify the impact of gRPC communication and serialization on overall performance
- **Scalability Testing**: Evaluate performance under concurrent load with multiple CPU cores and gRPC connections

### Architecture Comparison
- **Baseline (Local)**: Establish performance ceiling without network or API overhead
- **API Layer**: Measure FSC View API overhead for local node operations
- **gRPC Layer**: Quantify network communication costs in distributed deployments
- **Real-World Simulation**: Test actual deployment scenarios with separate client/server nodes

### Optimization Guidance
- **Bottleneck Identification**: Determine whether performance is limited by cryptography, network, or service logic
- **Configuration Tuning**: Find optimal settings for CPU cores, gRPC connections, and GOGC values
- **Deployment Planning**: Inform decisions about node placement, network topology, and resource allocation

## Quickstart

### 1. Local View Benchmark

Tests transfer validation directly in-process without any network or API overhead. This establishes the performance baseline.

```bash
cd token/core/zkatdlog/nogh/v1/validator/bench/transfer_service/
GOGC=10000 go test -run ^$ -bench=BenchmarkLocalTransferService -benchtime=30s -count=5 -cpu=1,4,8,16,32,64
```

**Parameters:**
- `GOGC=10000`: Reduces garbage collection frequency for more stable measurements
- `-benchtime=30s`: Run each benchmark for 30 seconds
- `-count=5`: Repeat each benchmark 5 times for statistical significance
- `-cpu=1,4,8,16,32,64`: Test with different numbers of parallel goroutines

### 2. API Benchmark (Local Node)

Tests transfer validation through FSC's View API on a single node. Measures API overhead compared to direct validation.

```bash
cd token/core/zkatdlog/nogh/v1/validator/bench/transfer_service/
GOGC=10000 go test -bench=BenchmarkAPI -benchtime=30s -count=5 -cpu=32
```

**What it does:**
- Spins up a temporary FSC node with View API
- Submits transfer validation requests through the API
- Measures end-to-end latency including API serialization

### 3. gRPC API Benchmark (Local)

Tests transfer validation through gRPC client-server architecture on localhost. Measures network serialization overhead.

```bash
cd token/core/zkatdlog/nogh/v1/validator/bench/transfer_service/
GOGC=10000 go test -bench=BenchmarkAPIGRPC -benchtime=30s -count=5 -cpu=32 -numConn=1,2,4,8
```

**Parameters:**
- `-numConn=1,2,4,8`: Test with different numbers of gRPC client connections

**What it does:**
- Starts FSC node with gRPC server
- Creates gRPC client(s) connecting to localhost
- Measures latency including gRPC marshaling/unmarshaling and network stack

### 4. Distributed Two-Node Benchmark

For realistic deployment testing with separate client and server machines, see the detailed setup guide:

**[Setting Up Two-Node Benchmarking →](setting_up_nodes.md)**

This setup is ideal for:
- Testing real network latency and bandwidth constraints
- Evaluating performance on production-like infrastructure
- Measuring impact of geographic distribution
- Load testing with dedicated client machines

## Architecture

### Benchmark Layers

```
┌─────────────────────────────────────────────────────────────┐
│ Layer 4: Distributed (Client-Server)                        │
│ Location: server/ and client/ directories                   │
│ Purpose: Real-world deployment simulation                   │
│ Network: Actual network between machines                    │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│ Layer 3: gRPC API (BenchmarkAPIGRPC)                        │
│ Purpose: Network serialization overhead measurement         │
│ Network: Localhost gRPC (127.0.0.1)                         │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│ Layer 2: View API (BenchmarkAPI)                            │
│ Purpose: FSC API overhead measurement                       │
│ Network: In-process API calls                               │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│ Layer 1: Direct (BenchmarkLocalTransferService)             │
│ Purpose: Baseline performance without overhead              │
│ Network: None (direct function calls)                       │
└─────────────────────────────────────────────────────────────┘
```

### View Pool Pattern

All benchmarks use a **view pool** to ensure realistic testing:

```go
type viewPool struct {
    views []view.View  // Pre-generated transfer views
    idx   atomic.Int64 // Round-robin index
}
```

**Why?** Without a pool, benchmarks would repeatedly verify the *same* ZK proof, which doesn't reflect real-world scenarios where each transaction is unique. The pool rotates through 64 different pre-generated proofs (by default) to simulate realistic validation workload.

## Understanding Results

### Metrics Reported

Each benchmark reports:
- **ns/op**: Nanoseconds per operation (lower is better)
- **TPS**: Transactions per second (higher is better)
- **B/op**: Bytes allocated per operation
- **allocs/op**: Number of allocations per operation

### Example Output

```
BenchmarkLocalTransferService/out-tokens=2in-tokens=2-32    1000    30000000 ns/op    33333 TPS
```

**Interpretation:**
- Test ran with 32 parallel goroutines (`-cpu=32`)
- 2 input tokens, 2 output tokens
- Each validation took ~30ms (30,000,000 ns)
- Achieved ~33,333 transactions per second

## Related Documentation

- **[Core Driver Benchmarks](core/dlognogh/dlognogh.md)** - Lower-level cryptographic benchmarks
- **[Testing Architecture](core/dlognogh/dlognogh_architecture.md)** - Understanding the test layers
- **[Setting Up Nodes](setting_up_nodes.md)** - Distributed two-node setup guide
- **[Benchmark Tools](tools.md)** - Analysis and profiling tools
- **[AWS Setup Guide](../../token/core/zkatdlog/nogh/v1/validator/bench/transfer_service/aws_setup_2_machines.md)** - Cloud deployment instructions

