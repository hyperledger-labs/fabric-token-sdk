# ZKAT DLog No Graph Hiding Benchmark

Packages with benchmark tests:

- `token/core/zkatdlog/nogh/v1/transfer`: `BenchmarkSender`, `BenchmarkVerificationSenderProof`, `TestParallelBenchmarkSender`
- `token/core/zkatdlog/nogh/v1/issue`: `BenchmarkIssuer` and `BenchmarkProofVerificationIssuer`
- `token/core/zkatdlog/nogh/v1`: `BenchmarkTransfer` 

The steps necessary to run the benchmarks are very similar.
We give two examples here:
- `token/core/zkatdlog/nogh/v1/transfer#BenchmarkSender`, and
- `token/core/zkatdlog/nogh/v1/transfer#TestParallelBenchmarkSender`

## Benchmark: `token/core/zkatdlog/nogh/v1/transfer#BenchmarkSender`

In this Section, we go through the steps necessary to run the benchmark and interpreter the results.
For the other benchmarks the process is the same.

### Overview

`BenchmarkSender` measures the cost of generating a zero-knowledge transfer (ZK transfer) using the DLog no-graph-hiding sender implementation and serializing the resulting transfer object. 
Concretely, each benchmark iteration constructs the required sender environment, invokes `GenerateZKTransfer(...)`, and calls `Serialize()` on the returned transfer — so the measured time includes ZK transfer construction and serialization.

The benchmark is implemented to run the same workload across a matrix of parameters (bit sizes, curve choices, number of inputs and outputs). 
A helper inside the test (`generateBenchmarkCases`) programmatically generates all combinations of the selected parameters.

### Parameters

The benchmark accepts the following tunable parameters (configured from the command line):

- Bits: integer bit sizes used for some setup (e.g., 32, 64). This is passed to the test setup code.
- CurveID: the `math.CurveID` used (examples: `BN254`, `BLS12_381_BBS_GURVY`).
- NumInputs: number of input tokens provided to the sender (1, 2, ...).
- NumOutputs: number of outputs produced by the transfer (1, 2, ...).

### Default parameter set used in the benchmark

The test file currently uses the following parameter slices (so the resulting combinations are the Cartesian product of these lists):

- bits: [32, 64]
- curves: [BN254, BLS12_381_BBS_GURVY, BLS12_381_BBS_GURVY_FAST_RNG]
- inputs: [1, 2, 3]
- outputs: [1, 2, 3]

This produces 2 (bits) * 3 (curves) * 3 (inputs) * 3 (outputs) = 54 sub-benchmarks. 
Each sub-benchmark runs the standard `b.N` iterations and reports time and allocation statistics.

### How to run

Run the benchmark for the package containing the sender benchmarks:

```sh
# run the BenchmarkSender benchmarks in the transfer package
go test ./token/core/zkatdlog/nogh/v1/transfer -bench=BenchmarkSender -benchmem -count=1 -cpu=1 -timeout 0 -run=^$
```
> Notice that:
> - `-run=^$` has the effect to avoid running any other unit-test present in the package.
> - `-timeout 0` disables the test timeout.

If you want to run the benchmark repeatedly and save results to a file:

```sh
go test ./token/core/zkatdlog/nogh/v1/transfer -bench=BenchmarkSender -benchmem -count=10 -cpu=1 -timeout 0 -run=^$ | tee bench.txt
```

Note: `-count` controls how many times the test binary is executed (useful to reduce variance); `-benchmem` reports allocation statistics.

You can also change the parameters:

```shell
go test ./token/core/zkatdlog/nogh/v1/transfer -test.bench=BenchmarkSender -test.benchmem -test.count=10 -test.cpu=1 -test.timeout 0 -test.run=^$ -bits="32" -curves="BN254" -num_inputs="2" -num_outputs="2"  | tee bench.txt
```

> Notice that in this the above case, the `go test` options must be prefixed with `test.` otherwise the tool will fail.
 
The test supports the following flags:

```shell
  -bits string
        a comma-separated list of bit sizes (32, 64,...)
  -curves string
        comma-separated list of curves. Supported curves are: FP256BN_AMCL, BN254, FP256BN_AMCL_MIRACL, BLS12_381_BBS, BLS12_381_BBS_GURVY, BLS12_381_BBS_GURVY_FAST_RNG
  -num_inputs string
        a comma-separate list of number of inputs (1,2,3,...)
  -num_outputs string
        a comma-separate list of number of outputs (1,2,3,...)
```

### Notes and best practices

- Be mindful of the Cartesian explosion: combining many bit sizes, curves, input counts and output counts can produce many sub-benchmarks.  
  For CI or quick local runs, reduce the parameter lists to a small subset (for example: one bit size, one curve, and 1-2 input/output sizes).
- The benchmark creates `b.N` independent sender environments (via `NewBenchmarkSenderEnv`) and runs `GenerateZKTransfer` for each environment in the inner loop — so memory and setup cost scale with `b.N` during setup.
- If you need to measure only the transfer-generation time and omit setup, consider modifying the benchmark to move expensive one-time setup out of the measured region and call `b.ResetTimer()` appropriately (the current benchmark already calls `b.ResetTimer()` before the inner loop).

### Collecting and interpreting results

A typical run prints timings per sub-benchmark (ns/op) and allocation statistics. Example command to persist results:

```sh
go test ./token/core/zkatdlog/nogh/v1/transfer -bench=BenchmarkSender -benchmem -count=10 -cpu=1 -timeout 0 -run=^$ | tee bench.txt
```

You can then aggregate/parse the output (e.g., benchstat) to compute averages across `-count` repetitions.

### Results

Example results have been produced on an Apple M1 Max and can be consulted [here](./transfer_BenchmarkSender_results.md). 

## Benchmark: `token/core/zkatdlog/nogh/v1/transfer#TestParallelBenchmarkSender`

This is a test that runs multiple instances of the above benchmark in parallel.
This allows the analyst to understand if shared data structures are actual bottlenecks.

```shell
go test ./token/core/zkatdlog/nogh/v1/transfer -test.run=TestParallelBenchmarkSender -test.v -test.benchmem -test.timeout 0 -bits="32" -curves="BN254" -num_inputs="2" -num_outputs="2" -workers="1,10" -duration="10s" | tee bench.txt
```

The test supports the following flags:
```shell
  -bits string
        a comma-separated list of bit sizes (32, 64,...)
  -curves string
        comma-separated list of curves. Supported curves are: FP256BN_AMCL, BN254, FP256BN_AMCL_MIRACL, BLS12_381_BBS, BLS12_381_BBS_GURVY, BLS12_381_BBS_GURVY_FAST_RNG
  -duration duration
        test duration (1s, 1m, 1h,...) (default 1s)
  -num_inputs string
        a comma-separate list of number of inputs (1,2,3,...)
  -num_outputs string
        a comma-separate list of number of outputs (1,2,3,...)
  -workers string
        a comma-separate list of workers (1,2,3,...,NumCPU), where NumCPU is converted to the number of available CPUs
```

### Results

```shell
=== RUN   TestParallelBenchmarkSender
=== RUN   TestParallelBenchmarkSender/Setup(bits_32,_curve_BN254,_#i_2,_#o_2)_with_1_workers
Metric           Value            Description
------           -----            -----------
Workers          1                
Total Ops        166              Total completions
Real Throughput  16.57/s          Ops/sec observed (includes setup overhead)
Pure Throughput  17.56/s          Theoretical Max Ops/sec (if setup was 0ms)
Avg Latency      56.934508ms      Time spent inside work()
Memory           1225816 B/op     Allocated bytes per operation
Allocs           17582 allocs/op  Allocations per operation
=== RUN   TestParallelBenchmarkSender/Setup(bits_32,_curve_BN254,_#i_2,_#o_2)_with_10_workers
Metric           Value            Description
------           -----            -----------
Workers          10               
Total Ops        1205             Total completions
Real Throughput  119.72/s         Ops/sec observed (includes setup overhead)
Pure Throughput  127.50/s         Theoretical Max Ops/sec (if setup was 0ms)
Avg Latency      78.430613ms      Time spent inside work()
Memory           1225512 B/op     Allocated bytes per operation
Allocs           17546 allocs/op  Allocations per operation
--- PASS: TestParallelBenchmarkSender (20.33s)
    --- PASS: TestParallelBenchmarkSender/Setup(bits_32,_curve_BN254,_#i_2,_#o_2)_with_1_workers (10.14s)
    --- PASS: TestParallelBenchmarkSender/Setup(bits_32,_curve_BN254,_#i_2,_#o_2)_with_10_workers (10.18s)
PASS
ok      github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/transfer       20.864s
```