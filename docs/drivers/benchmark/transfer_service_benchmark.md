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

To setup AWS EC2 nodes See: [AWS Benchmark 2 Machines](../../../token/core/zkatdlog/nogh/v1/validator/bench/transfer_service/aws_bench_2_machines.md)

For realistic deployment testing with separate client and server machines:

Architecture: 
1. Machine 1 is the server (FC Node)
2. Machine 2 is the client sending to the server and gathering the metrics

Setup: 
1. Add ssh pubkey of server to client `known_hosts` (If not already connected)  
2. Start the server:

```bash
cd token/core/zkatdlog/nogh/v1/validator/bench/transfer_service/
GOGC=10000 go run ./server/
```
Wait until you see the output:
```bash
Running fscnode test-node
```
[Note: We set GOGC to 10K but you don't have to]

3. In the client, copy the node data from the server  
We can use `Rsync` for this:
```bash
cd token/core/zkatdlog/nogh/v1/validator/bench/transfer_service/

rsync -avz <YourServerName>:/<fullpath>/fabric-token-sdk/token/core/zkatdlog/nogh/v1/validator/bench/transfer_service/out/ ./out
``` 
Note: Make sure full path and server name are correct

4. Replace the IP (or full name DNS can resolve) to the Server IP in the client out folder:

```bash
sed 's#127.0.0.1#123.456.789#g' ./out/testdata/fsc/nodes/test-node.0/client-config.yaml -i
```
5. Start client and tee output to file

```bash
GOGC=10000 nohup go run ./client/ -benchtime=30s -count=5 -workloads=transfer-service -cpu=1,2,4,8,16,32,48,64 -numConn=1,2,4,8 2>&1 | tee out.txt &
```

This setup is ideal for:
- Testing real network latency and bandwidth constraints
- Evaluating performance on production-like infrastructure
- Measuring impact of geographic distribution
- Load testing with dedicated client machines
## Visualising the results 

0. Set up python environment:

```bash
python -m venv .venv 
source .venv/bin/activate
pip install streamlit pandas
```

1. Place the results in `.txt` files in a folder called `bench`.

2. Run:
```bash
streamlit run cmd/benchmarking/plotly_plot_node.py
```

## Real world example:
- Server: dectrust2.vpc.cloud9.ibm.com
- Client: dectrust1.vpc.cloud9.ibm.com

They both already have each-others SSH Keys, so I don't need to do it again.

On dectrust2, Server Runs:
```bash
cd token/core/zkatdlog/nogh/v1/validator/bench/transfer_service/
GOGC=10000 go run ./server/
```
I wait to see:

```bash
Running fscnode test-node
```

On dectrust1:
```bash
cd ~/effi/fabric-token-sdk/token/core/zkatdlog/nogh/v1/validator/bench/transfer_service/
rsync -avz root@dectrust2.vpc.cloud9.ibm.com:/root/effi/fabric-token-sdk/token/core/zkatdlog/nogh/v1/validator/bench/transfer_service/out/ ./out
sed 's#127.0.0.1#dectrust1.vpc.cloud9.ibm.com#g' ./out/testdata/fsc/nodes/test-node.0/client-config.yaml -i
GOGC=10000 nohup go run ./client/ -benchtime=30s -count=5 -workloads=zkp -cpu=1,2,4,8,16,32 -numConn=1,2,4,8 2>&1 | tee example-2node.txt &
```

I also run gRPC benchmark without the 2 node setup:

```bash
GOGC=10000 go test -bench=BenchmarkAPIGRPC -benchtime=30s -count=5 -cpu=32 -numConn=1,2,4,8 | tee example-grpc.txt
```

I now move the outputs to `bench` folder and run the visualisation:

```bash
mv example-2node.txt bench
mv example-grpc.txt
```

(Optional) I setup python `venv`:
```bash
python -m venv .venv 
source .venv/bin/activate
pip install streamlit pandas
```

Now I can run the visualisation:
```bash 
streamlit run cmd/benchmarking/plotly_plot_node.py 
```

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
- **[Benchmark Tools](tools.md)** - Analysis and profiling tools
- **[AWS Setup Guide](../../../token/core/zkatdlog/nogh/v1/validator/bench/transfer_service/aws_bench_2_machines.md)** - Cloud deployment instructions

