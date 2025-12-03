# ZKAT DLog No Graph Hiding Benchmark

Packages with benchmark tests:

- `token/core/zkatdlog/nogh/v1/transfer`: 
   - `BenchmarkSender`, `BenchmarkVerificationSenderProof`, `TestParallelBenchmarkSender`, and `TestParallelBenchmarkVerificationSenderProof` are used to benchmark the generation of a transfer action. This includes also the generation of ZK proof for a transfer operation.
   - `BenchmarkTransferProofGeneration`, `TestParallelBenchmarkTransferProofGeneration` are used to benchmark the generation of ZK proof alone. 
- `token/core/zkatdlog/nogh/v1/issue`: `BenchmarkIssuer` and `BenchmarkProofVerificationIssuer`
- `token/core/zkatdlog/nogh/v1`: `BenchmarkTransfer` 

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

Example results have been produced on an Apple M1 Max and can be consulted [here](./transfer_BenchmarkSender_results.md). 

## Benchmark: `token/core/zkatdlog/nogh/v1/transfer#TestParallelBenchmarkSender`

This is a test that runs multiple instances of the above benchmark in parallel.
This allows the analyst to understand if shared data structures are actual bottlenecks.

It uses a custom-made runner whose documentation can be found [here](../../../token/core/common/benchmark/runner.md).

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

```go
=== RUN   TestParallelBenchmarkSender
=== RUN   TestParallelBenchmarkSender/Setup(bits_32,_curve_BN254,_#i_2,_#o_2)_with_1_workers
Metric           Value          Description
------           -----          -----------
Workers          1              
Total Ops        168            (Low Sample Size)
Duration         10.023390959s  (Good Duration)
Real Throughput  16.76/s        Observed Ops/sec (Wall Clock)
Pure Throughput  17.77/s        Theoretical Max (Low Overhead)

Latency Distribution:
 Min           55.180375ms  
 P50 (Median)  55.945812ms  
 Average       56.290356ms  
 P95           58.108814ms  
 P99           58.758087ms  
 Max           59.089958ms  (Stable Tail)

Stability Metrics:
 Std Dev  898.087µs   
 IQR      1.383083ms  Interquartile Range
 Jitter   590.076µs   Avg delta per worker
 CV       1.60%       Excellent Stability (<5%)

Memory  1301420 B/op     Allocated bytes per operation
Allocs  18817 allocs/op  Allocations per operation

Latency Heatmap (Dynamic Range):
Range                     Freq  Distribution Graph
 55.180375ms-55.369563ms  17    █████████████████████████ (10.1%)
 55.369563ms-55.5594ms    18    ██████████████████████████ (10.7%)
 55.5594ms-55.749887ms    27    ████████████████████████████████████████ (16.1%)
 55.749887ms-55.941028ms  20    █████████████████████████████ (11.9%)
 55.941028ms-56.132824ms  13    ███████████████████ (7.7%)
 56.132824ms-56.325277ms  9     █████████████ (5.4%)
 56.325277ms-56.51839ms   4     █████ (2.4%)
 56.51839ms-56.712165ms   6     ████████ (3.6%)
 56.712165ms-56.906605ms  9     █████████████ (5.4%)
 56.906605ms-57.101711ms  13    ███████████████████ (7.7%)
 57.101711ms-57.297486ms  10    ██████████████ (6.0%)
 57.297486ms-57.493933ms  3     ████ (1.8%)
 57.493933ms-57.691053ms  3     ████ (1.8%)
 57.691053ms-57.888849ms  4     █████ (2.4%)
 57.888849ms-58.087323ms  3     ████ (1.8%)
 58.087323ms-58.286478ms  2     ██ (1.2%)
 58.286478ms-58.486315ms  2     ██ (1.2%)
 58.486315ms-58.686837ms  2     ██ (1.2%)
 58.686837ms-58.888047ms  2     ██ (1.2%)
 58.888047ms-59.089958ms  1     █ (0.6%)

--- Analysis & Recommendations ---
[WARN] Low sample size (168). Results may not be statistically significant. Run for longer.
[INFO] High Allocations (18817/op). This will trigger frequent GC cycles and increase Max Latency.
----------------------------------
=== RUN   TestParallelBenchmarkSender/Setup(bits_32,_curve_BN254,_#i_2,_#o_2)_with_10_workers
Metric           Value          Description
------           -----          -----------
Workers          10             
Total Ops        1232           (Low Sample Size)
Duration         10.070877291s  (Good Duration)
Real Throughput  122.33/s       Observed Ops/sec (Wall Clock)
Pure Throughput  130.12/s       Theoretical Max (Low Overhead)

Latency Distribution:
 Min           61.2545ms     
 P50 (Median)  75.461375ms   
 Average       76.852256ms   
 P95           93.50851ms    
 P99           106.198982ms  
 Max           144.872375ms  (Stable Tail)

Stability Metrics:
 Std Dev  9.28799ms    
 IQR      10.909229ms  Interquartile Range
 Jitter   9.755984ms   Avg delta per worker
 CV       12.09%       Moderate Variance (10-20%)

Memory  1282384 B/op     Allocated bytes per operation
Allocs  18668 allocs/op  Allocations per operation

Latency Heatmap (Dynamic Range):
Range                       Freq  Distribution Graph
 61.2545ms-63.948502ms      36    ███████ (2.9%)
 63.948502ms-66.760987ms    86    █████████████████ (7.0%)
 66.760987ms-69.697167ms    152   ███████████████████████████████ (12.3%)
 69.697167ms-72.762481ms    181   █████████████████████████████████████ (14.7%)
 72.762481ms-75.962609ms    195   ████████████████████████████████████████ (15.8%)
 75.962609ms-79.303481ms    179   ████████████████████████████████████ (14.5%)
 79.303481ms-82.791286ms    152   ███████████████████████████████ (12.3%)
 82.791286ms-86.432486ms    94    ███████████████████ (7.6%)
 86.432486ms-90.233828ms    59    ████████████ (4.8%)
 90.233828ms-94.202355ms    40    ████████ (3.2%)
 94.202355ms-98.345419ms    29    █████ (2.4%)
 98.345419ms-102.670697ms   9     █ (0.7%)
 102.670697ms-107.186203ms  8     █ (0.6%)
 107.186203ms-111.900303ms  4      (0.3%)
 111.900303ms-116.821732ms  2      (0.2%)
 116.821732ms-121.959608ms  3      (0.2%)
 121.959608ms-127.32345ms   1      (0.1%)
 127.32345ms-132.923196ms   1      (0.1%)
 138.769222ms-144.872375ms  1      (0.1%)

--- Analysis & Recommendations ---
[WARN] Low sample size (1232). Results may not be statistically significant. Run for longer.
[INFO] High Allocations (18668/op). This will trigger frequent GC cycles and increase Max Latency.
----------------------------------
--- PASS: TestParallelBenchmarkSender (20.83s)
    --- PASS: TestParallelBenchmarkSender/Setup(bits_32,_curve_BN254,_#i_2,_#o_2)_with_1_workers (10.39s)
    --- PASS: TestParallelBenchmarkSender/Setup(bits_32,_curve_BN254,_#i_2,_#o_2)_with_10_workers (10.44s)
PASS
ok      github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/transfer       21.409s
```