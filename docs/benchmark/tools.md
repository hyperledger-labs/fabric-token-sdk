# Go Benchmark

This Section details how to execute Go benchmarks using the `go test` command and analyze the results with `benchstat`.

## 1. Running Benchmarks: Core Flags

To run benchmarks, use the `go test` command with specific flags. The most essential flag is `-bench`, which accepts a regular expression to select which benchmark functions to execute.

### Execution Control Flags

| Flag | Description | Example |
| :--- | :--- | :--- |
| `-bench=<regexp>` | Run benchmarks matching the regex. Use `.` to run all. | `-bench=.` |
| `-run=<regexp>` | Run unit tests matching the regex. Use `^$` to skip all unit tests and only run benchmarks. | `-run=^$` |
| `-benchtime=<t>` | Duration to run each benchmark (default `1s`). Can also specify iterations (suffix `x`). | `-benchtime=5s` or `-benchtime=1000x` |
| `-count=<n>` | Run each benchmark `n` times. Essential for statistical analysis with `benchstat`. | `-count=10` |
| `-timeout=<t>` | Overrides the default 10m timeout. Necessary for long-running benchmark suites. | `-timeout=30m` |
| `-failfast` | Stop execution immediately after the first failure. | `-failfast` |

### Resource & Profiling Flags

These flags are critical for performance tuning and memory analysis, which aligns with your recent work on allocation optimization.

| Flag | Description | Example |
| :--- | :--- | :--- |
| `-benchmem` | Print memory allocation statistics (allocations per op, bytes per op). | `-benchmem` |
| `-cpu=<n,m...>` | Run benchmarks with specific `GOMAXPROCS` values. | `-cpu=1,2,4,8` |
| `-cpuprofile=<file>` | Write a CPU profile to the specified file. | `-cpuprofile=cpu.out` |
| `-memprofile=<file>` | Write a memory profile to the specified file. | `-memprofile=mem.out` |
| `-blockprofile=<file>`| Write a goroutine blocking profile (contention analysis). | `-blockprofile=block.out` |
| `-mutexprofile=<file>`| Write a mutex contention profile. | `-mutexprofile=mutex.out` |
| `-trace=<file>` | Write an execution trace for the `go tool trace` viewer. | `-trace=trace.out` |

> **Note:** When using profiling flags like `-cpuprofile`, the resulting binary is preserved in the current directory for analysis with `go tool pprof`.

## 2. The Analysis Workflow (A/B Testing)

Reliable performance optimization requires measuring the "before" and "after" states. The standard workflow involves saving benchmark output to files and comparing them.

### Step 1: Install Benchstat

`benchstat` is the standard tool for computing statistical summaries and comparing Go benchmarks.
```bash
go install golang.org/x/perf/cmd/benchstat@latest
```
or

```shell
makefile install-tools
```

### Step 2: Capture Baseline (Old)

Run the current code 10 times to gather statistically significant data.
```bash
# Save current performance to old.txt
go test -bench=. -benchmem -count=10 -run=^$ . > old.txt
```

### Step 3: Capture Experiment (New)

Apply your code changes (e.g., the allocation optimizations you researched) and run the same benchmark command.
```bash
# Save optimized performance to new.txt
go test -bench=. -benchmem -count=10 -run=^$ . > new.txt
```

## 3. Analyzing Results with Benchstat

Run `benchstat` against your two captured files to see the delta.

```bash
benchstat old.txt new.txt
```

### Interpreting the Output

The output displays the mean value for each metric and the percentage change.

```text
name           old time/op    new time/op    delta
JSONEncode-8     1.50µs ± 2%    1.20µs ± 1%  -20.00%  (p=0.000 n=10+10)

name           old alloc/op   new alloc/op   delta
JSONEncode-8       896B ± 0%      420B ± 0%  -53.12%  (p=0.000 n=10+10)
```

*   **delta:** The percentage change. Negative values indicate improvement (reduced time or allocations).
*   **p-value:** The probability that the difference is due to random noise. A value `< 0.05` is generally considered statistically significant.
*   **n=10+10:** Indicates 10 valid samples were used from both the old and new files.

### Advanced Grouping

If your benchmarks use configuration naming conventions (e.g., `Benchmark/Enc=json` vs `Benchmark/Enc=gob`), you can group results specifically.

```bash
# Compare results grouping by a specific configuration key
benchstat -col /Enc old.txt
```

## 4. Best Practices for Accurate Results

* **Isolation:** Close high-CPU applications (browsers, IDE indexing) before running benchmarks to reduce noise.
*   **Count > 1:** Always use `-count` (ideally 5-10) to detect variance. Single runs are unreliable for optimization decisions.
*   **Run Filter:** Always use `-run=^$` to prevent unit tests from interfering with benchmark timing or output parsing.
*   **Stable Machine:** For mission-critical measurements, consider using a dedicated bare-metal machine or a cloud instance with pinned CPUs to avoid "noisy neighbor" effects.