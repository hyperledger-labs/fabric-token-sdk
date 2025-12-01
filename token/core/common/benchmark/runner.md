# Go Benchmark Runner

This package provides a robust, high-precision benchmark runner for Go applications. 
It goes beyond standard Go testing benchmarks by offering detailed statistical analysis, memory profiling, latency distribution visualization, and automated health checks.

## Overview

The `benchmark` package is designed to measure the performance of a specific operation (the "unit of work") under concurrent load. 
It orchestrates the execution of this work across multiple goroutines, captures precise latency data, and generates a comprehensive report.

## Key Features

*   **Concurrent Execution**: Runs the benchmark across a user-defined number of workers (goroutines).
*   **Two-Phase Measurement**:
    *   **Phase 1 (Memory)**: Runs serially in isolation to accurately measure heap allocations per operation without concurrency noise.
    *   **Phase 2 (Throughput/Latency)**: Runs concurrently to measure real-world throughput and latency distribution.
*   **Low-Overhead Recording**: Uses pre-allocated memory chunks to record latency data, ensuring the measuring process itself doesn't trigger Garbage Collection (GC) or skew results.
*   **Statistical Rigor**: Calculates advanced metrics like Interquartile Range (IQR), Jitter, Coefficient of Variation (CV), and interpolated percentiles.
*   **Visual Output**: Generates an ASCII-based latency heatmap and color-coded status indicators directly in the terminal.
*   **Automated Analysis**: Provides "Analysis & Recommendations" at the end of the run, flagging issues like high variance, GC pressure, or unstable tail latencies.

## Usage

The core entry point is the generic `RunBenchmark` function.

### Function Signature

```go
func RunBenchmark[T any](
    workers int,                  // Number of concurrent goroutines
    benchDuration time.Duration,  // How long to run the test
    setup func() T,               // Function to prepare data for each op
    work func(T),                 // The function to benchmark
) Result
```

### Example

```go
package main

import (
    "time"
    "your/package/benchmark" // Import the runner
)

func main() {
    // Define the benchmark
    result := benchmark.RunBenchmark(
        10,                 // 10 concurrent workers
        5*time.Second,      // Run for 5 seconds
        func() int {        // Setup: Prepare data (not timed)
            return 42 
        },
        func(input int) {   // Work: The operation to measure (timed)
            // Simulate work
            process(input)
        },
    )

    // Print the detailed report to stdout
    result.Print()
}
```

## Output Breakdown

The `Result.Print()` method outputs a structured report divided into several sections:

### 1. Main Metrics
Basic throughput and volume statistics.
*   **Total Ops**: Total number of operations completed.
*   **Real Throughput**: Observed operations per second (Wall Clock).
*   **Pure Throughput**: Theoretical maximum throughput (excludes overhead).
*   **Overhead**: Calculates the percentage of time lost to coordination or setup costs.

### 2. Latency Distribution
A detailed look at how long operations took.
*   **P50, P95, P99**: Standard percentiles.
*   **Tail Latency Check**: Compares Max Latency to P99 to identify extreme outliers (e.g., "Max is 12x P99").

### 3. Stability Metrics
Measures how consistent the system is.
*   **Std Dev**: Standard deviation of latencies.
*   **IQR (Interquartile Range)**: A measure of spread that is robust against outliers (difference between P75 and P25).
*   **Jitter**: The average change in latency between consecutive operations on the same worker.
*   **CV (Coefficient of Variation)**: `StdDev / Mean`. Used to grade stability (e.g., <5% is "Excellent").

### 4. Memory
*   **Allocated**: Bytes allocated per operation.
*   **Allocs**: Number of heap allocations per operation.

### 5. Latency Heatmap
An ASCII bar chart visualizing the distribution of latencies. It uses color coding (Green/Yellow/Red) to indicate frequency density.

```text
Range           Freq    Distribution Graph
100µs-200µs     500     ██████ (5.0%)
200µs-400µs     8000    ██████████████████████ (80.0%)
...
```

### 6. Analysis & Recommendations
The runner automatically evaluates the results and prints warnings or pass/fail statuses:
*   **[WARN] Low sample size**: If total operations < 5000.
*   **[FAIL] High Variance**: If the Coefficient of Variation > 20%.
*   **[INFO] High Allocations**: If allocations > 100/op (warns about GC pressure).
*   **[CRITICAL] Massive Latency Spikes**: If the Max latency is significantly higher than the P99.

## Limitations & Considerations

While this runner is robust for general application testing, users should be aware of the following constraints:

### 1. Memory Consumption at Scale
The runner retains **every** latency sample in memory to provide precise percentiles (P99, P99.9) without approximation errors.
*   **Impact**: It consumes approximately **80MB of RAM per 10 million operations**.
*   **Constraint**: Running long "soak tests" (e.g., billions of operations) may trigger an Out-Of-Memory (OOM) crash.

### 2. "Setup" Function Blocking
The `setup()` function runs sequentially *inside* the worker loop.
*   **Impact**: While `setup()` time is excluded from *Latency* metrics, it is included in the *Real Throughput* calculation.
*   **Constraint**: If your `setup()` is slow (e.g., creates a complex object), you may see confusing results: low Latency (fast work) but low Throughput (slow loop).

### 3. Nanosecond Precision
The runner uses standard `time.Now()` and `time.Since()`.
*   **Constraint**: For operations taking less than **50-100ns** (like simple arithmetic), the overhead of the timer itself becomes significant. This runner is best suited for operations in the microsecond/millisecond range (e.g., DB queries, API calls, cryptographic functions).

### 4. Coarse Histogram
The visualization is hardcoded to 20 buckets using an exponential scale.
*   **Constraint**: You cannot "zoom in" to a specific latency range or change the bucket resolution via configuration.

### 5. Stop-Time Latency
The runner signals workers to stop using an atomic flag, but it waits for the current operation to finish.
*   **Constraint**: If a single operation hangs or takes minutes, the benchmark cannot stop immediately when the duration expires. It must wait for stragglers to complete.

## Implementation Details

### Zero-Allocation Recording
To prevent the benchmark from measuring itself, the runner uses a linked-list of fixed-size arrays (`chunk`).
*   **Structure**: `type chunk struct { data [10000]time.Duration; ... }`
*   **Benefit**: Recording a latency requires only a simple array index increment. No `append()` or slice growth occurs during the "hot path" of the benchmark.

### Histogram Calculation
It uses an **Exponential Histogram** (`calcExponentialHistogramImproved`) logic.
*   Buckets grow exponentially, allowing the runner to capture both very small (nanosecond) and very large (second) latencies in the same graph with high fidelity.
*   It handles boundary conditions precisely to avoid floating-point drift.