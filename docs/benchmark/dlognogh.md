# ZKAT DLog No Graph Hiding Benchmark

Packages with benchmark tests:

- `token/core/zkatdlog/nogh/v1/transfer`: `BenchmarkSender` and `BenchmarkSenderProofVerification`

## Benchmark: `token/core/zkatdlog/nogh/v1/transfer#BenchmarkSender`

### Overview

`BenchmarkSender` measures the cost of generating a zero-knowledge transfer (ZK transfer) using the DLog no-graph-hiding sender implementation and serializing the resulting transfer object. 
Concretely, each benchmark iteration constructs the required sender environment, invokes `GenerateZKTransfer(...)`, and calls `Serialize()` on the returned transfer — so the measured time includes ZK transfer construction and serialization.

The benchmark is implemented to run the same workload across a matrix of parameters (bit sizes, curve choices, number of inputs and outputs). 
A helper inside the test (`generateBenchmarkCases`) programmatically generates all combinations of the selected parameters.

### Parameters

The benchmark accepts the following tunable parameters (configured inside the test file):

- Bits: integer bit sizes used for some setup (e.g., 32, 64). This is passed to the test setup code.
- CurveID: the `math.CurveID` used (examples: `math.BN254`, `math.BLS12_381_BBS_GURVY`).
- NumInputs: number of input tokens provided to the sender (1, 2, ...).
- NumOutputs: number of outputs produced by the transfer (1, 2, ...).

These values are provided as slices inside `BenchmarkSender` and combined via `generateBenchmarkCases` to form the full set of sub-benchmarks.

### Default parameter set used in the benchmark

The test file currently uses the following example parameter slices (so the resulting combinations are the Cartesian product of these lists):

- bits: [32, 64]
- curves: [math.BN254, math.BLS12_381_BBS_GURVY]
- inputs: [1, 2, 3]
- outputs: [1, 2, 3]

This produces 2 (bits) * 2 (curves) * 3 (inputs) * 3 (outputs) = 36 sub-benchmarks. 
Each sub-benchmark runs the standard `b.N` iterations and reports time and allocation statistics.

### How to run

Run the benchmark for the package containing the sender benchmarks:

```sh
# run the BenchmarkSender benchmarks in the transfer package
go test ./token/core/zkatdlog/nogh/v1/transfer -bench=BenchmarkSender -benchmem -count=1 -cpu=1
```

If you want to run the benchmark repeatedly and save results to a file:

```sh
go test ./token/core/zkatdlog/nogh/v1/transfer -bench=BenchmarkSender -benchmem -count=10 -cpu=1 | tee bench.txt
```

Note: `-count` controls how many times the test binary is executed (useful to reduce variance); `-benchmem` reports allocation statistics.

### Running a single sub-case

`BenchmarkSender` creates sub-benchmarks with names like `Setup(bits 32, curve BN254, #i 1, #o 1)`. 
You can select a sub-benchmark by passing a regular expression to `-bench`. 
For example (the exact quoting may vary by shell):

```sh
# run only the sub-benchmarks whose name contains the substring "bits 32" and "BN254"
go test ./token/core/zkatdlog/nogh/v1/transfer -bench='BenchmarkSender/Setup\(bits 32, curve BN254' -benchmem
```

If the exact sub-benchmark name includes characters that are interpreted by the shell (parentheses, commas, spaces), quoting/escaping the regex is required — it's often easiest to choose a shorter substring that appears in the sub-benchmark name (for example, `BenchmarkSender/Setup` or `BenchmarkSender/Setup.*BN254`).

Alternatively, edit the slices in the test file to temporarily contain just the parameters you want to exercise.

### Notes and best practices

- Be mindful of the Cartesian explosion: combining many bit sizes, curves, input counts and output counts can produce many sub-benchmarks.  
  For CI or quick local runs, reduce the parameter lists to a small subset (for example: one bit size, one curve, and 1-2 input/output sizes).
- The benchmark creates `b.N` independent sender environments (via `NewBenchmarkSenderEnv`) and runs `GenerateZKTransfer` for each environment in the inner loop — so memory and setup cost scale with `b.N` during setup.
- If you need to measure only the transfer-generation time and omit setup, consider modifying the benchmark to move expensive one-time setup out of the measured region and call `b.ResetTimer()` appropriately (the current benchmark already calls `b.ResetTimer()` before the inner loop).

### Collecting and interpreting results

A typical run prints timings per sub-benchmark (ns/op) and allocation statistics. Example command to persist results:

```sh
go test ./token/core/zkatdlog/nogh/v1/transfer -bench=BenchmarkSender -benchmem -count=10 -cpu=1 -timeout 0 -run=^ | tee bench.txt
```

> Notice that: 
> - `-run=^` has the effect to avoid running any other unit-test present in the package.
> - `-timeout 0` disables the test timeout.

You can then aggregate/parse the output (e.g., benchstat) to compute averages across `-count` repetitions.

### Results

```shell
goos: darwin
goarch: arm64
pkg: github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/transfer
cpu: Apple M1 Max
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_1,_#o_1)         	    2032	    588969 ns/op	   24095 B/op	     436 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_1,_#o_1)         	    2102	    586849 ns/op	   24095 B/op	     436 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_1,_#o_1)         	    2103	    569845 ns/op	   24095 B/op	     436 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_1,_#o_1)         	    2136	    572235 ns/op	   24096 B/op	     436 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_1,_#o_1)         	    2115	    572774 ns/op	   24095 B/op	     436 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_1,_#o_1)         	    2160	    579751 ns/op	   24095 B/op	     436 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_1,_#o_1)         	    2092	    579448 ns/op	   24095 B/op	     436 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_1,_#o_1)         	    2128	    611803 ns/op	   24095 B/op	     436 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_1,_#o_1)         	    2121	    604121 ns/op	   24096 B/op	     436 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_1,_#o_1)         	    2149	    571305 ns/op	   24095 B/op	     436 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_1,_#o_2)         	      20	  57604733 ns/op	 1220495 B/op	   17507 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_1,_#o_2)         	      19	  63136362 ns/op	 1220557 B/op	   17512 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_1,_#o_2)         	      20	  59511919 ns/op	 1220498 B/op	   17506 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_1,_#o_2)         	      20	  58289552 ns/op	 1220457 B/op	   17502 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_1,_#o_2)         	      20	  61169150 ns/op	 1220484 B/op	   17506 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_1,_#o_2)         	      20	  58029802 ns/op	 1220421 B/op	   17502 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_1,_#o_2)         	      20	  58154669 ns/op	 1220532 B/op	   17512 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_1,_#o_2)         	      20	  61878602 ns/op	 1220466 B/op	   17508 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_1,_#o_2)         	      19	  58902158 ns/op	 1220534 B/op	   17509 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_1,_#o_2)         	      20	  61470454 ns/op	 1220492 B/op	   17507 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_1,_#o_3)         	      13	  89144862 ns/op	 1820095 B/op	   26068 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_1,_#o_3)         	      13	  88252285 ns/op	 1819937 B/op	   26056 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_1,_#o_3)         	      13	  87102740 ns/op	 1819930 B/op	   26051 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_1,_#o_3)         	      13	  86766654 ns/op	 1819878 B/op	   26054 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_1,_#o_3)         	      13	  86701513 ns/op	 1819861 B/op	   26049 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_1,_#o_3)         	      13	  87121551 ns/op	 1819996 B/op	   26065 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_1,_#o_3)         	      13	  96531654 ns/op	 1819892 B/op	   26059 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_1,_#o_3)         	      13	  87142369 ns/op	 1820019 B/op	   26066 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_1,_#o_3)         	      13	  87067994 ns/op	 1819945 B/op	   26056 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_1,_#o_3)         	      13	  87193551 ns/op	 1820040 B/op	   26064 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_2,_#o_1)         	      39	  29643584 ns/op	  625691 B/op	    9025 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_2,_#o_1)         	      39	  32067403 ns/op	  625635 B/op	    9025 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_2,_#o_1)         	      40	  29357174 ns/op	  625707 B/op	    9028 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_2,_#o_1)         	      40	  29297232 ns/op	  625722 B/op	    9032 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_2,_#o_1)         	      40	  29351622 ns/op	  625703 B/op	    9028 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_2,_#o_1)         	      42	  29405333 ns/op	  625709 B/op	    9029 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_2,_#o_1)         	      40	  29338427 ns/op	  625677 B/op	    9027 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_2,_#o_1)         	      40	  30347148 ns/op	  625674 B/op	    9027 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_2,_#o_1)         	      40	  29430769 ns/op	  625682 B/op	    9025 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_2,_#o_1)         	      40	  29334336 ns/op	  625673 B/op	    9026 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_2,_#o_2)         	      20	  64167112 ns/op	 1224543 B/op	   17571 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_2,_#o_2)         	      19	  60353044 ns/op	 1224647 B/op	   17580 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_2,_#o_2)         	      20	  58291462 ns/op	 1224604 B/op	   17575 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_2,_#o_2)         	      19	  58761182 ns/op	 1224545 B/op	   17571 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_2,_#o_2)         	      20	  58298990 ns/op	 1224615 B/op	   17576 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_2,_#o_2)         	      20	  58199156 ns/op	 1224569 B/op	   17572 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_2,_#o_2)         	      20	  57951823 ns/op	 1224599 B/op	   17573 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_2,_#o_2)         	      20	  60568902 ns/op	 1224652 B/op	   17577 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_2,_#o_2)         	      20	  58267052 ns/op	 1224611 B/op	   17573 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_2,_#o_2)         	      20	  58223762 ns/op	 1224704 B/op	   17584 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_2,_#o_3)         	      13	  86805837 ns/op	 1824328 B/op	   26122 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_2,_#o_3)         	      13	  88718260 ns/op	 1824295 B/op	   26121 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_2,_#o_3)         	      12	  87670160 ns/op	 1824286 B/op	   26126 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_2,_#o_3)         	      12	 105300795 ns/op	 1824378 B/op	   26129 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_2,_#o_3)         	      13	  88846923 ns/op	 1824184 B/op	   26113 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_2,_#o_3)         	      13	  87306263 ns/op	 1824206 B/op	   26114 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_2,_#o_3)         	      13	  87348708 ns/op	 1824272 B/op	   26117 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_2,_#o_3)         	      13	  91839734 ns/op	 1824271 B/op	   26124 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_2,_#o_3)         	      13	  86474115 ns/op	 1824328 B/op	   26125 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_2,_#o_3)         	      13	  86665253 ns/op	 1824263 B/op	   26120 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_3,_#o_1)         	      40	  29656133 ns/op	  631403 B/op	    9106 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_3,_#o_1)         	      39	  29709674 ns/op	  631396 B/op	    9105 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_3,_#o_1)         	      40	  31194578 ns/op	  631412 B/op	    9107 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_3,_#o_1)         	      39	  31395681 ns/op	  631428 B/op	    9108 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_3,_#o_1)         	      39	  31239450 ns/op	  631414 B/op	    9106 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_3,_#o_1)         	      40	  29552445 ns/op	  631416 B/op	    9107 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_3,_#o_1)         	      40	  29593196 ns/op	  631432 B/op	    9106 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_3,_#o_1)         	      39	  29700333 ns/op	  631466 B/op	    9110 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_3,_#o_1)         	      39	  30977215 ns/op	  631432 B/op	    9108 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_3,_#o_1)         	      39	  30885849 ns/op	  631449 B/op	    9106 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_3,_#o_2)         	      20	  58374683 ns/op	 1231692 B/op	   17659 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_3,_#o_2)         	      20	  58233875 ns/op	 1231700 B/op	   17661 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_3,_#o_2)         	      20	  58245435 ns/op	 1231720 B/op	   17661 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_3,_#o_2)         	      20	  59514206 ns/op	 1231725 B/op	   17666 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_3,_#o_2)         	      20	  59163477 ns/op	 1231752 B/op	   17667 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_3,_#o_2)         	      20	  58289821 ns/op	 1231768 B/op	   17667 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_3,_#o_2)         	      19	  58483989 ns/op	 1231657 B/op	   17654 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_3,_#o_2)         	      19	  58564958 ns/op	 1231700 B/op	   17661 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_3,_#o_2)         	      20	  58504460 ns/op	 1231782 B/op	   17668 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_3,_#o_2)         	      20	  58322706 ns/op	 1231718 B/op	   17662 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_3,_#o_3)         	      13	  87896240 ns/op	 1831956 B/op	   26217 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_3,_#o_3)         	      13	  87711917 ns/op	 1831984 B/op	   26220 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_3,_#o_3)         	      13	  87992894 ns/op	 1831960 B/op	   26216 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_3,_#o_3)         	      13	  87264641 ns/op	 1831967 B/op	   26216 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_3,_#o_3)         	      13	  87106080 ns/op	 1831919 B/op	   26212 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_3,_#o_3)         	      13	  86356237 ns/op	 1831976 B/op	   26221 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_3,_#o_3)         	      13	  90977292 ns/op	 1831900 B/op	   26207 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_3,_#o_3)         	      13	  90689208 ns/op	 1831952 B/op	   26220 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_3,_#o_3)         	      13	 101240359 ns/op	 1831953 B/op	   26217 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BN254,_#i_3,_#o_3)         	      13	  89385949 ns/op	 1831847 B/op	   26217 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_1)         	    2138	    575613 ns/op	   24095 B/op	     436 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_1)         	    2182	    570220 ns/op	   24095 B/op	     436 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_1)         	    2176	    575743 ns/op	   24095 B/op	     436 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_1)         	    2131	    579320 ns/op	   24094 B/op	     436 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_1)         	    2118	    578257 ns/op	   24095 B/op	     436 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_1)         	    2136	    590342 ns/op	   24095 B/op	     436 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_1)         	    2124	    576329 ns/op	   24094 B/op	     436 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_1)         	    2108	    577738 ns/op	   24095 B/op	     436 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_1)         	    2095	    574212 ns/op	   24096 B/op	     436 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_1)         	    2097	    574390 ns/op	   24095 B/op	     436 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_2)         	      20	  58664360 ns/op	 1220530 B/op	   17512 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_2)         	      20	  58277490 ns/op	 1220484 B/op	   17509 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_2)         	      20	  57937781 ns/op	 1220543 B/op	   17512 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_2)         	      20	  58187583 ns/op	 1220472 B/op	   17508 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_2)         	      20	  58296056 ns/op	 1220438 B/op	   17503 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_2)         	      20	  58213294 ns/op	 1220506 B/op	   17513 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_2)         	      20	  57868031 ns/op	 1220497 B/op	   17508 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_2)         	      20	  57816846 ns/op	 1220474 B/op	   17503 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_2)         	      20	  57669219 ns/op	 1220509 B/op	   17512 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_2)         	      20	  57704042 ns/op	 1220449 B/op	   17501 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_3)         	      13	  86855163 ns/op	 1820069 B/op	   26065 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_3)         	      13	  86300256 ns/op	 1819953 B/op	   26060 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_3)         	      12	  86583490 ns/op	 1820053 B/op	   26056 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_3)         	      13	  86385785 ns/op	 1819972 B/op	   26052 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_3)         	      13	  86458808 ns/op	 1819883 B/op	   26055 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_3)         	      13	  86880660 ns/op	 1819952 B/op	   26056 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_3)         	      13	  87096000 ns/op	 1819910 B/op	   26053 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_3)         	      13	  86848561 ns/op	 1819973 B/op	   26059 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_3)         	      13	  87306865 ns/op	 1820044 B/op	   26067 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_3)         	      13	  86715272 ns/op	 1819918 B/op	   26053 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_1)         	      40	  29249093 ns/op	  625694 B/op	    9029 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_1)         	      39	  29512885 ns/op	  625646 B/op	    9026 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_1)         	      40	  29190820 ns/op	  625709 B/op	    9028 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_1)         	      40	  29298525 ns/op	  625679 B/op	    9029 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_1)         	      40	  29248554 ns/op	  625710 B/op	    9029 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_1)         	      40	  29159949 ns/op	  625690 B/op	    9027 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_1)         	      40	  29301975 ns/op	  625659 B/op	    9024 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_1)         	      40	  29285148 ns/op	  625744 B/op	    9032 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_1)         	      40	  29264357 ns/op	  625699 B/op	    9028 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_1)         	      40	  29540347 ns/op	  625660 B/op	    9025 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_2)         	      20	  58304971 ns/op	 1224659 B/op	   17580 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_2)         	      20	  57977690 ns/op	 1224538 B/op	   17570 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_2)         	      20	  58186744 ns/op	 1224579 B/op	   17571 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_2)         	      20	  58252588 ns/op	 1224624 B/op	   17573 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_2)         	      20	  58157738 ns/op	 1224646 B/op	   17577 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_2)         	      20	  58427140 ns/op	 1224611 B/op	   17574 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_2)         	      19	  58310596 ns/op	 1224530 B/op	   17572 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_2)         	      20	  58203071 ns/op	 1224658 B/op	   17580 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_2)         	      20	  58296408 ns/op	 1224578 B/op	   17576 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_2)         	      20	  58135025 ns/op	 1224633 B/op	   17576 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_3)         	      13	  91864718 ns/op	 1824320 B/op	   26120 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_3)         	      13	  87007622 ns/op	 1824345 B/op	   26121 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_3)         	      13	  87767574 ns/op	 1824243 B/op	   26121 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_3)         	      13	  87476570 ns/op	 1824324 B/op	   26124 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_3)         	      13	  87051077 ns/op	 1824329 B/op	   26128 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_3)         	      13	  86963596 ns/op	 1824369 B/op	   26129 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_3)         	      13	  87077170 ns/op	 1824298 B/op	   26124 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_3)         	      13	  87438128 ns/op	 1824334 B/op	   26122 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_3)         	      13	  87213808 ns/op	 1824388 B/op	   26130 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_3)         	      13	  86966343 ns/op	 1824352 B/op	   26123 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_1)         	      39	  29601099 ns/op	  631437 B/op	    9107 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_1)         	      39	  29671221 ns/op	  631406 B/op	    9105 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_1)         	      39	  29541880 ns/op	  631473 B/op	    9112 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_1)         	      40	  29632855 ns/op	  631413 B/op	    9108 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_1)         	      39	  29618970 ns/op	  631446 B/op	    9109 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_1)         	      40	  31387192 ns/op	  631415 B/op	    9106 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_1)         	      39	  29955197 ns/op	  631447 B/op	    9110 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_1)         	      39	  29861400 ns/op	  631448 B/op	    9108 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_1)         	      40	  29624452 ns/op	  631422 B/op	    9106 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_1)         	      39	  29724272 ns/op	  631415 B/op	    9108 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_2)         	      20	  58735079 ns/op	 1231780 B/op	   17668 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_2)         	      20	  58723646 ns/op	 1231718 B/op	   17665 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_2)         	      20	  58390146 ns/op	 1231677 B/op	   17658 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_2)         	      20	  59086633 ns/op	 1231789 B/op	   17670 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_2)         	      20	  58667785 ns/op	 1231684 B/op	   17659 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_2)         	      20	  58738912 ns/op	 1231762 B/op	   17665 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_2)         	      20	  58544471 ns/op	 1231697 B/op	   17660 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_2)         	      20	  59094958 ns/op	 1231739 B/op	   17664 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_2)         	      20	  60461817 ns/op	 1231721 B/op	   17663 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_2)         	      19	  59154029 ns/op	 1231642 B/op	   17660 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_3)         	      13	  87766071 ns/op	 1831969 B/op	   26218 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_3)         	      13	  87426324 ns/op	 1831987 B/op	   26222 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_3)         	      13	  90921490 ns/op	 1831802 B/op	   26209 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_3)         	      13	  86806032 ns/op	 1832026 B/op	   26223 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_3)         	      13	  87204657 ns/op	 1831733 B/op	   26206 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_3)         	      13	  87770487 ns/op	 1831947 B/op	   26222 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_3)         	      13	  87724718 ns/op	 1831886 B/op	   26211 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_3)         	      13	  87408638 ns/op	 1831852 B/op	   26210 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_3)         	      13	  87892131 ns/op	 1831899 B/op	   26210 allocs/op
BenchmarkSender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_3)         	      13	  87983215 ns/op	 1832049 B/op	   26222 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_1,_#o_1)                       	    2134	    581202 ns/op	   24095 B/op	     436 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_1,_#o_1)                       	    2043	    616811 ns/op	   24095 B/op	     436 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_1,_#o_1)                       	    2092	    629385 ns/op	   24095 B/op	     436 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_1,_#o_1)                       	    2160	    576581 ns/op	   24095 B/op	     436 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_1,_#o_1)                       	    2222	    584641 ns/op	   24095 B/op	     436 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_1,_#o_1)                       	    2132	    551651 ns/op	   24096 B/op	     436 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_1,_#o_1)                       	    2193	    546922 ns/op	   24094 B/op	     436 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_1,_#o_1)                       	    2238	    547594 ns/op	   24095 B/op	     436 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_1,_#o_1)                       	    2193	    550651 ns/op	   24095 B/op	     436 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_1,_#o_1)                       	    2252	    548710 ns/op	   24095 B/op	     436 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_1,_#o_2)                       	      10	 112308000 ns/op	 2321807 B/op	   33128 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_1,_#o_2)                       	       9	 111490065 ns/op	 2321687 B/op	   33110 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_1,_#o_2)                       	      10	 110510433 ns/op	 2321688 B/op	   33113 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_1,_#o_2)                       	      10	 109454979 ns/op	 2321804 B/op	   33123 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_1,_#o_2)                       	       9	 114398519 ns/op	 2321601 B/op	   33107 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_1,_#o_2)                       	      10	 109766612 ns/op	 2321844 B/op	   33131 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_1,_#o_2)                       	      10	 109822754 ns/op	 2321784 B/op	   33126 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_1,_#o_2)                       	      10	 109814588 ns/op	 2321671 B/op	   33107 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_1,_#o_2)                       	      10	 109699179 ns/op	 2321887 B/op	   33134 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_1,_#o_2)                       	      10	 110221700 ns/op	 2321596 B/op	   33100 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_1,_#o_3)                       	       7	 168809399 ns/op	 3472417 B/op	   49481 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_1,_#o_3)                       	       7	 169757083 ns/op	 3472617 B/op	   49486 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_1,_#o_3)                       	       7	 163849048 ns/op	 3472485 B/op	   49483 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_1,_#o_3)                       	       7	 167993637 ns/op	 3472582 B/op	   49476 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_1,_#o_3)                       	       6	 166674542 ns/op	 3472482 B/op	   49481 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_1,_#o_3)                       	       7	 169427929 ns/op	 3472321 B/op	   49467 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_1,_#o_3)                       	       7	 167872262 ns/op	 3472501 B/op	   49479 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_1,_#o_3)                       	       7	 169105863 ns/op	 3472385 B/op	   49474 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_1,_#o_3)                       	       7	 163753542 ns/op	 3472536 B/op	   49490 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_1,_#o_3)                       	       7	 164771708 ns/op	 3472422 B/op	   49465 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_2,_#o_1)                       	      21	  55075254 ns/op	 1175994 B/op	   16828 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_2,_#o_1)                       	      21	  55540593 ns/op	 1176059 B/op	   16834 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_2,_#o_1)                       	      21	  55130135 ns/op	 1176124 B/op	   16838 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_2,_#o_1)                       	      21	  55059827 ns/op	 1175995 B/op	   16827 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_2,_#o_1)                       	      20	  56088108 ns/op	 1176007 B/op	   16831 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_2,_#o_1)                       	      21	  55345202 ns/op	 1176012 B/op	   16829 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_2,_#o_1)                       	      20	  56055927 ns/op	 1175974 B/op	   16826 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_2,_#o_1)                       	      20	  55932512 ns/op	 1176217 B/op	   16853 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_2,_#o_1)                       	      21	  55195823 ns/op	 1176138 B/op	   16842 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_2,_#o_1)                       	      21	  55013075 ns/op	 1176058 B/op	   16832 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_2,_#o_2)                       	      10	 109379908 ns/op	 2325879 B/op	   33192 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_2,_#o_2)                       	      10	 109358196 ns/op	 2325855 B/op	   33188 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_2,_#o_2)                       	      10	 109548979 ns/op	 2325700 B/op	   33177 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_2,_#o_2)                       	      10	 109734167 ns/op	 2325719 B/op	   33173 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_2,_#o_2)                       	      10	 110554788 ns/op	 2325880 B/op	   33185 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_2,_#o_2)                       	      10	 109379438 ns/op	 2325807 B/op	   33188 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_2,_#o_2)                       	      10	 109471788 ns/op	 2325847 B/op	   33184 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_2,_#o_2)                       	      10	 109333892 ns/op	 2325756 B/op	   33173 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_2,_#o_2)                       	      10	 112020188 ns/op	 2325874 B/op	   33189 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_2,_#o_2)                       	      10	 110189300 ns/op	 2325871 B/op	   33190 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_2,_#o_3)                       	       7	 163925946 ns/op	 3476330 B/op	   49562 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_2,_#o_3)                       	       7	 163971464 ns/op	 3475805 B/op	   49523 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_2,_#o_3)                       	       7	 163697768 ns/op	 3475755 B/op	   49514 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_2,_#o_3)                       	       7	 164190809 ns/op	 3475748 B/op	   49513 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_2,_#o_3)                       	       7	 171872488 ns/op	 3475776 B/op	   49509 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_2,_#o_3)                       	       7	 164836917 ns/op	 3476153 B/op	   49556 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_2,_#o_3)                       	       7	 164707655 ns/op	 3475905 B/op	   49520 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_2,_#o_3)                       	       7	 170578911 ns/op	 3475888 B/op	   49532 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_2,_#o_3)                       	       6	 169069750 ns/op	 3475906 B/op	   49519 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_2,_#o_3)                       	       6	 173323694 ns/op	 3475672 B/op	   49505 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_3,_#o_1)                       	      20	  55549827 ns/op	 1182167 B/op	   16914 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_3,_#o_1)                       	      21	  55168163 ns/op	 1182152 B/op	   16913 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_3,_#o_1)                       	      21	  55217692 ns/op	 1182150 B/op	   16910 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_3,_#o_1)                       	      21	  55402833 ns/op	 1182140 B/op	   16908 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_3,_#o_1)                       	      21	  55686585 ns/op	 1182202 B/op	   16916 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_3,_#o_1)                       	      21	  55343784 ns/op	 1182074 B/op	   16901 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_3,_#o_1)                       	      21	  57085679 ns/op	 1182182 B/op	   16911 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_3,_#o_1)                       	      21	  55400256 ns/op	 1182110 B/op	   16907 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_3,_#o_1)                       	      21	  55374579 ns/op	 1182106 B/op	   16904 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_3,_#o_1)                       	      21	  56957962 ns/op	 1182079 B/op	   16907 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_3,_#o_2)                       	      10	 112694683 ns/op	 2332524 B/op	   33270 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_3,_#o_2)                       	       9	 111462370 ns/op	 2332418 B/op	   33260 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_3,_#o_2)                       	      10	 110743967 ns/op	 2332500 B/op	   33267 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_3,_#o_2)                       	      10	 109985783 ns/op	 2332539 B/op	   33273 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_3,_#o_2)                       	      10	 109836742 ns/op	 2332756 B/op	   33293 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_3,_#o_2)                       	      10	 113944217 ns/op	 2332590 B/op	   33272 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_3,_#o_2)                       	      10	 110206712 ns/op	 2332521 B/op	   33260 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_3,_#o_2)                       	      10	 114075162 ns/op	 2332741 B/op	   33288 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_3,_#o_2)                       	       9	 112472194 ns/op	 2332582 B/op	   33281 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_3,_#o_2)                       	      10	 109760958 ns/op	 2332420 B/op	   33253 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_3,_#o_3)                       	       7	 165579601 ns/op	 3483857 B/op	   49636 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_3,_#o_3)                       	       7	 164970821 ns/op	 3483811 B/op	   49621 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_3,_#o_3)                       	       7	 167979798 ns/op	 3483818 B/op	   49625 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_3,_#o_3)                       	       6	 176933729 ns/op	 3483634 B/op	   49605 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_3,_#o_3)                       	       7	 165346173 ns/op	 3483778 B/op	   49634 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_3,_#o_3)                       	       7	 165094309 ns/op	 3484019 B/op	   49646 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_3,_#o_3)                       	       7	 165162934 ns/op	 3483996 B/op	   49641 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_3,_#o_3)                       	       7	 164890619 ns/op	 3483657 B/op	   49604 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_3,_#o_3)                       	       7	 170351875 ns/op	 3483884 B/op	   49636 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BN254,_#i_3,_#o_3)                       	       7	 164672774 ns/op	 3483884 B/op	   49621 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_1)         	    2226	    548504 ns/op	   24094 B/op	     436 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_1)         	    2245	    551112 ns/op	   24095 B/op	     436 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_1)         	    2222	    555199 ns/op	   24095 B/op	     436 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_1)         	    2175	    556495 ns/op	   24094 B/op	     436 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_1)         	    2194	    552308 ns/op	   24096 B/op	     436 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_1)         	    2188	    575518 ns/op	   24094 B/op	     436 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_1)         	    2161	    569070 ns/op	   24095 B/op	     436 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_1)         	    2020	    577794 ns/op	   24095 B/op	     436 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_1)         	    2186	    580815 ns/op	   24094 B/op	     436 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_1)         	    2066	    573671 ns/op	   24095 B/op	     436 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_2)         	       9	 112831537 ns/op	 2321799 B/op	   33126 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_2)         	      10	 111305621 ns/op	 2321762 B/op	   33119 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_2)         	      10	 115915483 ns/op	 2321758 B/op	   33120 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_2)         	       9	 111846329 ns/op	 2321917 B/op	   33137 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_2)         	      10	 113429758 ns/op	 2321624 B/op	   33113 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_2)         	       9	 113834824 ns/op	 2321635 B/op	   33115 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_2)         	       9	 112761921 ns/op	 2321601 B/op	   33106 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_2)         	      10	 121665600 ns/op	 2321815 B/op	   33128 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_2)         	       9	 112720468 ns/op	 2321685 B/op	   33119 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_2)         	       9	 112249991 ns/op	 2321827 B/op	   33125 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_3)         	       6	 168178868 ns/op	 3472682 B/op	   49502 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_3)         	       6	 168030924 ns/op	 3472149 B/op	   49438 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_3)         	       6	 167709764 ns/op	 3472236 B/op	   49455 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_3)         	       6	 168071611 ns/op	 3472504 B/op	   49477 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_3)         	       6	 175927320 ns/op	 3472266 B/op	   49458 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_3)         	       6	 168407180 ns/op	 3472496 B/op	   49481 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_3)         	       6	 167976049 ns/op	 3472288 B/op	   49469 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_3)         	       6	 168844542 ns/op	 3472424 B/op	   49466 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_3)         	       6	 169188458 ns/op	 3472433 B/op	   49484 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_3)         	       6	 174894202 ns/op	 3472442 B/op	   49474 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_1)         	      20	  56414269 ns/op	 1176065 B/op	   16834 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_1)         	      20	  56370538 ns/op	 1175962 B/op	   16821 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_1)         	      20	  56326035 ns/op	 1176018 B/op	   16830 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_1)         	      20	  56933562 ns/op	 1175974 B/op	   16827 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_1)         	      20	  56393867 ns/op	 1176066 B/op	   16836 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_1)         	      20	  56424108 ns/op	 1176025 B/op	   16833 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_1)         	      20	  56282623 ns/op	 1176006 B/op	   16827 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_1)         	      20	  56203092 ns/op	 1176057 B/op	   16833 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_1)         	      20	  56383292 ns/op	 1176015 B/op	   16829 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_1)         	      21	  56318179 ns/op	 1176040 B/op	   16829 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_2)         	       9	 112388051 ns/op	 2325919 B/op	   33195 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_2)         	       9	 113246852 ns/op	 2325858 B/op	   33186 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_2)         	       9	 112764903 ns/op	 2325739 B/op	   33177 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_2)         	       9	 112729898 ns/op	 2325905 B/op	   33185 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_2)         	       9	 112365218 ns/op	 2325914 B/op	   33185 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_2)         	       9	 112439389 ns/op	 2325985 B/op	   33192 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_2)         	       9	 112060926 ns/op	 2325691 B/op	   33175 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_2)         	       9	 115982579 ns/op	 2325995 B/op	   33202 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_2)         	       9	 111913120 ns/op	 2325780 B/op	   33187 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_2)         	       9	 111969486 ns/op	 2325814 B/op	   33185 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_3)         	       6	 167872924 ns/op	 3476026 B/op	   49532 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_3)         	       6	 167507729 ns/op	 3475837 B/op	   49508 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_3)         	       6	 168143403 ns/op	 3476066 B/op	   49545 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_3)         	       6	 167985625 ns/op	 3476213 B/op	   49564 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_3)         	       6	 168984708 ns/op	 3476256 B/op	   49573 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_3)         	       6	 170212535 ns/op	 3475856 B/op	   49530 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_3)         	       6	 168870243 ns/op	 3475936 B/op	   49531 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_3)         	       6	 168222583 ns/op	 3475722 B/op	   49506 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_3)         	       6	 168033958 ns/op	 3475912 B/op	   49523 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_3)         	       6	 167848840 ns/op	 3475990 B/op	   49554 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_1)         	      20	  56615381 ns/op	 1182229 B/op	   16918 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_1)         	      20	  58129488 ns/op	 1182165 B/op	   16911 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_1)         	      20	  56487002 ns/op	 1182146 B/op	   16911 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_1)         	      20	  56624365 ns/op	 1182127 B/op	   16911 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_1)         	      20	  56581694 ns/op	 1182088 B/op	   16905 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_1)         	      20	  56602325 ns/op	 1182220 B/op	   16919 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_1)         	      20	  56508573 ns/op	 1182138 B/op	   16909 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_1)         	      20	  56641333 ns/op	 1182185 B/op	   16913 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_1)         	      20	  57134727 ns/op	 1182112 B/op	   16906 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_1)         	      20	  56988515 ns/op	 1182131 B/op	   16907 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_2)         	       9	 112890537 ns/op	 2332613 B/op	   33269 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_2)         	       9	 112552454 ns/op	 2332623 B/op	   33279 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_2)         	       9	 112552593 ns/op	 2332415 B/op	   33255 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_2)         	       9	 112479963 ns/op	 2332624 B/op	   33273 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_2)         	       9	 112304792 ns/op	 2332449 B/op	   33259 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_2)         	       9	 112580204 ns/op	 2332529 B/op	   33261 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_2)         	       9	 112547801 ns/op	 2332684 B/op	   33287 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_2)         	       9	 112789060 ns/op	 2332508 B/op	   33269 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_2)         	       9	 112251250 ns/op	 2332569 B/op	   33275 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_2)         	       9	 112359509 ns/op	 2332616 B/op	   33275 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_3)         	       6	 171169757 ns/op	 3484045 B/op	   49653 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_3)         	       6	 168886430 ns/op	 3483890 B/op	   49636 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_3)         	       6	 169997070 ns/op	 3483781 B/op	   49624 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_3)         	       6	 180483486 ns/op	 3484013 B/op	   49645 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_3)         	       6	 171722847 ns/op	 3483794 B/op	   49630 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_3)         	       6	 167921104 ns/op	 3483632 B/op	   49609 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_3)         	       6	 168692368 ns/op	 3483974 B/op	   49629 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_3)         	       6	 168405514 ns/op	 3483545 B/op	   49613 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_3)         	       6	 169331424 ns/op	 3483881 B/op	   49641 allocs/op
BenchmarkSender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_3)         	       6	 168059111 ns/op	 3483745 B/op	   49632 allocs/op
PASS
ok  	github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/transfer	472.265s

```

Here is the summary produced by `benchstat`. 

```shell
benchstat bench.txt 
goos: darwin
goarch: arm64
pkg: github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/transfer
cpu: Apple M1 Max
                                                             │  bench.txt  │
                                                             │   sec/op    │
Sender/Setup(bits_32,_curve_BN254,_#i_1,_#o_1)                 579.6µ ± 4%
Sender/Setup(bits_32,_curve_BN254,_#i_1,_#o_2)                 59.21m ± 5%
Sender/Setup(bits_32,_curve_BN254,_#i_1,_#o_3)                 87.13m ± 2%
Sender/Setup(bits_32,_curve_BN254,_#i_2,_#o_1)                 29.38m ± 3%
Sender/Setup(bits_32,_curve_BN254,_#i_2,_#o_2)                 58.30m ± 4%
Sender/Setup(bits_32,_curve_BN254,_#i_2,_#o_3)                 87.51m ± 5%
Sender/Setup(bits_32,_curve_BN254,_#i_3,_#o_1)                 30.30m ± 3%
Sender/Setup(bits_32,_curve_BN254,_#i_3,_#o_2)                 58.43m ± 1%
Sender/Setup(bits_32,_curve_BN254,_#i_3,_#o_3)                 87.94m ± 3%
Sender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_1)   576.0µ ± 1%
Sender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_2)   58.06m ± 1%
Sender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_3)   86.78m ± 0%
Sender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_1)   29.27m ± 1%
Sender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_2)   58.23m ± 0%
Sender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_3)   87.15m ± 1%
Sender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_1)   29.65m ± 1%
Sender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_2)   58.74m ± 1%
Sender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_3)   87.75m ± 1%
Sender/Setup(bits_64,_curve_BN254,_#i_1,_#o_1)                 564.1µ ± 9%
Sender/Setup(bits_64,_curve_BN254,_#i_1,_#o_2)                 110.0m ± 2%
Sender/Setup(bits_64,_curve_BN254,_#i_1,_#o_3)                 167.9m ± 2%
Sender/Setup(bits_64,_curve_BN254,_#i_2,_#o_1)                 55.27m ± 1%
Sender/Setup(bits_64,_curve_BN254,_#i_2,_#o_2)                 109.5m ± 1%
Sender/Setup(bits_64,_curve_BN254,_#i_2,_#o_3)                 164.8m ± 4%
Sender/Setup(bits_64,_curve_BN254,_#i_3,_#o_1)                 55.40m ± 3%
Sender/Setup(bits_64,_curve_BN254,_#i_3,_#o_2)                 111.1m ± 3%
Sender/Setup(bits_64,_curve_BN254,_#i_3,_#o_3)                 165.3m ± 3%
Sender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_1)   562.8µ ± 3%
Sender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_2)   112.8m ± 3%
Sender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_3)   168.3m ± 4%
Sender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_1)   56.38m ± 0%
Sender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_2)   112.4m ± 1%
Sender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_3)   168.1m ± 1%
Sender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_1)   56.62m ± 1%
Sender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_2)   112.6m ± 0%
Sender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_3)   169.1m ± 2%
geomean                                                        45.75m

                                                             │   bench.txt   │
                                                             │     B/op      │
Sender/Setup(bits_32,_curve_BN254,_#i_1,_#o_1)                  23.53Ki ± 0%
Sender/Setup(bits_32,_curve_BN254,_#i_1,_#o_2)                  1.164Mi ± 0%
Sender/Setup(bits_32,_curve_BN254,_#i_1,_#o_3)                  1.736Mi ± 0%
Sender/Setup(bits_32,_curve_BN254,_#i_2,_#o_1)                  611.0Ki ± 0%
Sender/Setup(bits_32,_curve_BN254,_#i_2,_#o_2)                  1.168Mi ± 0%
Sender/Setup(bits_32,_curve_BN254,_#i_2,_#o_3)                  1.740Mi ± 0%
Sender/Setup(bits_32,_curve_BN254,_#i_3,_#o_1)                  616.6Ki ± 0%
Sender/Setup(bits_32,_curve_BN254,_#i_3,_#o_2)                  1.175Mi ± 0%
Sender/Setup(bits_32,_curve_BN254,_#i_3,_#o_3)                  1.747Mi ± 0%
Sender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_1)    23.53Ki ± 0%
Sender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_2)    1.164Mi ± 0%
Sender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_3)    1.736Mi ± 0%
Sender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_1)    611.0Ki ± 0%
Sender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_2)    1.168Mi ± 0%
Sender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_3)    1.740Mi ± 0%
Sender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_1)    616.6Ki ± 0%
Sender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_2)    1.175Mi ± 0%
Sender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_3)    1.747Mi ± 0%
Sender/Setup(bits_64,_curve_BN254,_#i_1,_#o_1)                  23.53Ki ± 0%
Sender/Setup(bits_64,_curve_BN254,_#i_1,_#o_2)                  2.214Mi ± 0%
Sender/Setup(bits_64,_curve_BN254,_#i_1,_#o_3)                  3.312Mi ± 0%
Sender/Setup(bits_64,_curve_BN254,_#i_2,_#o_1)                  1.122Mi ± 0%
Sender/Setup(bits_64,_curve_BN254,_#i_2,_#o_2)                  2.218Mi ± 0%
Sender/Setup(bits_64,_curve_BN254,_#i_2,_#o_3)                  3.315Mi ± 0%
Sender/Setup(bits_64,_curve_BN254,_#i_3,_#o_1)                  1.127Mi ± 0%
Sender/Setup(bits_64,_curve_BN254,_#i_3,_#o_2)                  2.224Mi ± 0%
Sender/Setup(bits_64,_curve_BN254,_#i_3,_#o_3)                  3.322Mi ± 0%
Sender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_1)    23.53Ki ± 0%
Sender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_2)    2.214Mi ± 0%
Sender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_3)    3.312Mi ± 0%
Sender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_1)    1.122Mi ± 0%
Sender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_2)    2.218Mi ± 0%
Sender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_3)    3.315Mi ± 0%
Sender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_1)    1.127Mi ± 0%
Sender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_2)    2.225Mi ± 0%
Sender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_3)    3.322Mi ± 0%
geomean                                                        1011.8Ki

                                                             │  bench.txt  │
                                                             │  allocs/op  │
Sender/Setup(bits_32,_curve_BN254,_#i_1,_#o_1)                  436.0 ± 0%
Sender/Setup(bits_32,_curve_BN254,_#i_1,_#o_2)                 17.51k ± 0%
Sender/Setup(bits_32,_curve_BN254,_#i_1,_#o_3)                 26.06k ± 0%
Sender/Setup(bits_32,_curve_BN254,_#i_2,_#o_1)                 9.027k ± 0%
Sender/Setup(bits_32,_curve_BN254,_#i_2,_#o_2)                 17.57k ± 0%
Sender/Setup(bits_32,_curve_BN254,_#i_2,_#o_3)                 26.12k ± 0%
Sender/Setup(bits_32,_curve_BN254,_#i_3,_#o_1)                 9.107k ± 0%
Sender/Setup(bits_32,_curve_BN254,_#i_3,_#o_2)                 17.66k ± 0%
Sender/Setup(bits_32,_curve_BN254,_#i_3,_#o_3)                 26.22k ± 0%
Sender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_1)    436.0 ± 0%
Sender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_2)   17.51k ± 0%
Sender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_3)   26.06k ± 0%
Sender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_1)   9.028k ± 0%
Sender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_2)   17.57k ± 0%
Sender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_3)   26.12k ± 0%
Sender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_1)   9.108k ± 0%
Sender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_2)   17.66k ± 0%
Sender/Setup(bits_32,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_3)   26.21k ± 0%
Sender/Setup(bits_64,_curve_BN254,_#i_1,_#o_1)                  436.0 ± 0%
Sender/Setup(bits_64,_curve_BN254,_#i_1,_#o_2)                 33.12k ± 0%
Sender/Setup(bits_64,_curve_BN254,_#i_1,_#o_3)                 49.48k ± 0%
Sender/Setup(bits_64,_curve_BN254,_#i_2,_#o_1)                 16.83k ± 0%
Sender/Setup(bits_64,_curve_BN254,_#i_2,_#o_2)                 33.19k ± 0%
Sender/Setup(bits_64,_curve_BN254,_#i_2,_#o_3)                 49.52k ± 0%
Sender/Setup(bits_64,_curve_BN254,_#i_3,_#o_1)                 16.91k ± 0%
Sender/Setup(bits_64,_curve_BN254,_#i_3,_#o_2)                 33.27k ± 0%
Sender/Setup(bits_64,_curve_BN254,_#i_3,_#o_3)                 49.63k ± 0%
Sender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_1)    436.0 ± 0%
Sender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_2)   33.12k ± 0%
Sender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_1,_#o_3)   49.47k ± 0%
Sender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_1)   16.83k ± 0%
Sender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_2)   33.19k ± 0%
Sender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_2,_#o_3)   49.53k ± 0%
Sender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_1)   16.91k ± 0%
Sender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_2)   33.27k ± 0%
Sender/Setup(bits_64,_curve_BLS12_381_BBS_GURVY,_#i_3,_#o_3)   49.63k ± 0%
geomean                                                        15.22k
```

