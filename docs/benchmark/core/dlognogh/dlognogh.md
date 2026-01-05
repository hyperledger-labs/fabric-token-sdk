# ZKAT DLog No Graph Hiding Benchmarks

Packages with benchmark tests:

- `token/core/zkatdlog/nogh/v1/transfer`: 
   - `BenchmarkSender`, `BenchmarkVerificationSenderProof`, `TestParallelBenchmarkSender`, and `TestParallelBenchmarkVerificationSenderProof` are used to benchmark the generation of a transfer action, serialization included. This includes also the generation of ZK proof for a transfer operation.
   - `BenchmarkTransferProofGeneration`, `TestParallelBenchmarkTransferProofGeneration` are used to benchmark the generation of ZK proof alone. This includes proof serialization. 
- `token/core/zkatdlog/nogh/v1/issue`: `BenchmarkIssuer` and `BenchmarkProofVerificationIssuer`
- `token/core/zkatdlog/nogh/v1/validator`: `TestParallelBenchmarkValidatorTransfer`.
- `token/core/zkatdlog/nogh/v1`: `BenchmarkTransferServiceTransfer` and `TestParallelBenchmarkTransferServiceTransfer`.

The steps necessary to run the benchmarks are very similar.
We give two examples here:
- `token/core/zkatdlog/nogh/v1/transfer#BenchmarkSender`, and
- `token/core/zkatdlog/nogh/v1/transfer#TestParallelBenchmarkSender`

## Benchmark: `token/core/zkatdlog/nogh/v1/transfer#BenchmarkSender`

In this Section, we go through the steps necessary to run the benchmark and interpret the results.
For the other benchmarks the process is the same.

### Overview

`BenchmarkSender` measures the cost of generating a zero-knowledge transfer (ZK transfer) using the DLog no-graph-hiding sender implementation and serializing the resulting transfer object. 
Concretely, each benchmark iteration constructs the required sender environment, invokes `GenerateZKTransfer(...)`, and calls `Serialize()` on the returned transfer - so the measured time includes ZK transfer construction and serialization.

The benchmark is implemented to run the same workload across a matrix of parameters (bit sizes, curve choices, number of inputs and outputs). 
A helper inside the test (`generateBenchmarkCases`) programmatically generates all combinations of the selected parameters.

### Parameters

The benchmark accepts the following tunable parameters:

- Bits: integer bit sizes used for some setup (e.g., 32, 64). This is passed to the test setup code.
- CurveID: the `math.CurveID` used (examples: `BN254`, `BLS12_381_BBS_GURVY`).
- NumInputs: number of input tokens provided to the sender (1, 2, ...).
- NumOutputs: number of outputs produced by the transfer (1, 2, ...).

These parameters can be configured from the command line using the following flags:

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

### Default parameter set used in the benchmark

If no flag is used, the test file currently uses the following parameter slices (so the resulting combinations are the Cartesian product of these lists):

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

Example results have been produced on an Apple M1 Max and can be consulted [here](transfer_BenchmarkSender_results.md). 

## Benchmark: `token/core/zkatdlog/nogh/v1/transfer#TestParallelBenchmarkSender`

This is a test that runs multiple instances of the above benchmark in parallel.
This allows the analyst to understand if shared data structures are actual bottlenecks.

It uses a custom-made runner whose documentation can be found [here](../../../../token/services/benchmark/runner.md).

```shell
go test ./token/core/zkatdlog/nogh/v1/transfer -test.run=TestParallelBenchmarkSender -test.v -test.timeout 0 -bits="32" -curves="BN254" -num_inputs="2" -num_outputs="2" -workers="NumCPU" -duration="10s" -setup_samples=128 | tee bench.txt
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
  -profile bool
        write pprof profiles to file
  -setup_samples uint
        number of setup samples, 0 disables it
```

### Results

```shell
=== RUN   TestParallelBenchmarkSender
=== RUN   TestParallelBenchmarkSender/Setup(bits_32,_curve_BN254,_#i_2,_#o_2)_with_10_workers
Metric           Value     Description
------           -----     -----------
Workers          10        
Total Ops        1230      (Low Sample Size)
Duration         10.068s   (Good Duration)
Real Throughput  122.17/s  Observed Ops/sec (Wall Clock)
Pure Throughput  123.04/s  Theoretical Max (Low Overhead)

Latency Distribution:
 Min           59.895916ms   
 P50 (Median)  77.717333ms   
 Average       81.27214ms    
 P95           112.28194ms   
 P99           137.126207ms  
 P99.9         189.117473ms  
 Max           215.981417ms  (Stable Tail)

Stability Metrics:
 Std Dev  16.96192ms   
 IQR      19.050834ms  Interquartile Range
 Jitter   15.937043ms  Avg delta per worker
 CV       20.87%       Unstable (>20%) - Result is Noisy

System Health & Reliability:
 Error Rate   0.0000%          (100% Success) (0 errors)
 Memory       1159374 B/op     Allocated bytes per operation
 Allocs       17213 allocs/op  Allocations per operation
 Alloc Rate   133.20 MB/s      Memory pressure on system
 GC Overhead  1.27%            (High GC Pressure)
 GC Pause     127.435871ms     Total Stop-The-World time
 GC Cycles    264              Full garbage collection cycles

Latency Heatmap (Dynamic Range):
Range                       Freq  Distribution Graph
 59.895916ms-63.862831ms    98    ██████████████████████ (8.0%)
 63.862831ms-68.092476ms    163   ████████████████████████████████████ (13.3%)
 68.092476ms-72.602251ms    170   ██████████████████████████████████████ (13.8%)
 72.602251ms-77.410709ms    172   ██████████████████████████████████████ (14.0%)
 77.410709ms-82.537631ms    177   ████████████████████████████████████████ (14.4%)
 82.537631ms-88.004111ms    128   ████████████████████████████ (10.4%)
 88.004111ms-93.832637ms    119   ██████████████████████████ (9.7%)
 93.832637ms-100.047186ms   73    ████████████████ (5.9%)
 100.047186ms-106.673326ms  40    █████████ (3.3%)
 106.673326ms-113.738317ms  32    ███████ (2.6%)
 113.738317ms-121.271222ms  20    ████ (1.6%)
 121.271222ms-129.303034ms  14    ███ (1.1%)
 129.303034ms-137.866793ms  12    ██ (1.0%)
 137.866793ms-146.997731ms  3      (0.2%)
 146.997731ms-156.733413ms  4      (0.3%)
 167.11389ms-178.181868ms   2      (0.2%)
 178.181868ms-189.98288ms   1      (0.1%)
 189.98288ms-202.565475ms   1      (0.1%)
 202.565475ms-215.981417ms  1      (0.1%)

--- Analysis & Recommendations ---
[WARN] Low sample size (1230). Results may not be statistically significant. Run for longer.
[FAIL] High Variance (CV 20.87%). System noise is affecting results. Isolate the machine or increase duration.
[INFO] High Allocations (17213/op). This will trigger frequent GC cycles and increase Max Latency.
----------------------------------

--- Throughput Timeline ---
Timeline: [▇▇▇█▇▇▇▇▆▇] (Max: 131 ops/s)

--- PASS: TestParallelBenchmarkSender (13.97s)
    --- PASS: TestParallelBenchmarkSender/Setup(bits_32,_curve_BN254,_#i_2,_#o_2)_with_10_workers (13.96s)
PASS
ok      github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/transfer       14.566s
```