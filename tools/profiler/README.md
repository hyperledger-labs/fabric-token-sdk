# Test & Benchmark Profiler

Automatic hierarchical call tracing and performance profiling for Go tests and benchmarks in the Token SDK.

## Quick Start

```bash
cd tools/profiler
./profile.sh BenchmarkValidatorTransfer
```

## What It Does

1. Copies your repo to `/tmp/profiler-<PID>` (your files are never touched)
2. Finds your test/benchmark in `token/` directory
3. Instruments all Go packages in `token/`
4. Runs with profiling
5. Saves results to current directory
6. Cleans up automatically

**Safety:** All work happens in `/tmp`. Your repository is never modified.

## Example Output

```
## Call Hierarchy

└── >>> (*Validator).VerifyTransfer [2.5s, 100.00%]
    ├── >>> (*Validator).verifyInputs [1.8s, 72.00%]
    │   └── >>> (*Verifier).Verify x5 [1.5s, 60.00%]
    └── (*Validator).verifyOutputs [0.7s, 28.00%]

## Top 20 Functions by Time

*Note: These functions are marked with `>>>` in the call tree above*

| Function | Total Time | % of Root |
|----------|------------|-----------|
| (*Validator).VerifyTransfer | 2.5s | 100.00% |
| (*Validator).verifyInputs | 1.8s | 72.00% |
| (*Verifier).Verify | 1.5s | 60.00% |
```

**Reading the output:**
- `>>>` marks the top 20 functions by time
- Times are **cumulative** (include all child functions)
- `x5` means called 5 times (total time shown)
- Percentages relative to root function

## Options

```bash
./profile.sh <test_or_benchmark_name> [options]

Options:
  -d, --display <mode>      Display: both, time, percent (default: both)
  -f, --root-function <name> Start from this function (show subtree)
  -m, --min-percent <n>     Hide functions below this % (default: 0)
  -h, --help                Show help
```

**Examples:**
```bash
# Show only percentages
./profile.sh BenchmarkValidatorTransfer -d percent

# Profile a subtree
./profile.sh BenchmarkValidatorTransfer -f VerifyTransfer

# Hide functions using less than 1%
./profile.sh BenchmarkValidatorTransfer -m 1.0

# Profile a test
./profile.sh TestValidatorTransfer
```

## Scope

**Searches for tests/benchmarks in:**
- `token/` directory and all subdirectories

**Instruments:**
- All Go packages in `token/` directory

**Output:**
- Saved to current directory as `<TestName>_profile.md`

## Files

- `profile.sh` - Automated profiling script
- `auto-instrument.go` - Adds tracer hooks to Go files
- `inject-tracer.go` - Injects tracer into test/benchmark
- `tracer/` - Runtime tracing library

## Troubleshooting

**Can't find test/benchmark:**
- Check spelling
- Ensure it's in `token/` directory

**Empty output file:**
- Check if test/benchmark runs successfully
- Look for compilation errors in script output

**Times don't add up:**
- Times are cumulative (include children)
- Use percentages to see relative contributions
