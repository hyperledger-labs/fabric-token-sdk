/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package benchmark

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"text/tabwriter"
	"time"
)

// Result holds the comprehensive benchmark metrics
type Result struct {
	GoRoutines    int
	OpsTotal      uint64
	Duration      time.Duration
	OpsPerSecReal float64 // Real throughput (including setup cost)
	OpsPerSecPure float64 // Theoretical max throughput (excluding setup)
	AvgLatency    time.Duration
	BytesPerOp    uint64
	AllocsPerOp   uint64
}

//nolint:errcheck
func (r Result) Print() {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Metric\tValue\tDescription")
	fmt.Fprintln(w, "------\t-----\t-----------")
	fmt.Fprintf(w, "Workers\t%d\t\n", r.GoRoutines)
	fmt.Fprintf(w, "Total Ops\t%d\tTotal completions\n", r.OpsTotal)
	fmt.Fprintf(w, "Real Throughput\t%.2f/s\tOps/sec observed (includes setup overhead)\n", r.OpsPerSecReal)
	fmt.Fprintf(w, "Pure Throughput\t%.2f/s\tTheoretical Max Ops/sec (if setup was 0ms)\n", r.OpsPerSecPure)
	fmt.Fprintf(w, "Avg Latency\t%v\tTime spent inside work()\n", r.AvgLatency)
	fmt.Fprintf(w, "Memory\t%d B/op\tAllocated bytes per operation\n", r.BytesPerOp)
	fmt.Fprintf(w, "Allocs\t%d allocs/op\tAllocations per operation\n", r.AllocsPerOp)
	w.Flush()
}

// RunBenchmark executes the setup and work functions concurrently
// T is the type of data passed from setup to work
func RunBenchmark[T any](
	workers int,
	benchDuration time.Duration,
	setup func() T,
	work func(T),
) Result {
	// ---------------------------------------------------------
	// PHASE 1: Memory Analysis (Serial & Isolated)
	// ---------------------------------------------------------
	// We measure memory serially because concurrent allocs are hard to attribute
	// strictly to 'work' vs 'setup' using global MemStats.

	// Warmup to stabilize heap
	warmupData := setup()
	work(warmupData)
	runtime.GC()

	var memBefore, memAfter runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	// Run once for measurement
	data := setup()

	// Snapshot *after* setup but *before* work to exclude setup allocs
	// Note: This assumes 'data' doesn't escape to heap in a way that complicates this,
	// but it's the best approximation without complex tracing.
	runtime.ReadMemStats(&memBefore)
	work(data)
	runtime.ReadMemStats(&memAfter)

	allocs := memAfter.Mallocs - memBefore.Mallocs
	bytes := memAfter.TotalAlloc - memBefore.TotalAlloc

	// ---------------------------------------------------------
	// PHASE 2: Throughput & Latency (Concurrent)
	// ---------------------------------------------------------
	var (
		opsCount      atomic.Uint64
		totalWorkTime atomic.Int64 // Cumulative time spent strictly in work()
		wg            sync.WaitGroup
	)

	startGlobal := time.Now()
	stopChan := make(chan struct{})

	// Spawn workers
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stopChan:
					return
				default:
					// 1. Setup (Excluded from timing)
					d := setup()

					// 2. Work (Measured)
					start := time.Now()
					work(d)
					duration := time.Since(start)

					// 3. Record metrics
					opsCount.Add(1)
					totalWorkTime.Add(int64(duration))
				}
			}
		}()
	}

	// Let it run
	time.Sleep(benchDuration)
	close(stopChan)
	wg.Wait()

	globalDuration := time.Since(startGlobal)
	totalOps := opsCount.Load()
	totalTimeNs := totalWorkTime.Load()

	// Calculations
	var avgLatency time.Duration
	var pureThroughput float64

	if totalOps > 0 {
		avgLatency = time.Duration(totalTimeNs / int64(totalOps))
		// Pure Throughput = How fast could we go if setup was instant?
		// Formula: Concurrency / AvgLatency
		if avgLatency > 0 {
			pureThroughput = float64(workers) / avgLatency.Seconds()
		}
	}

	return Result{
		GoRoutines:    workers,
		OpsTotal:      totalOps,
		Duration:      globalDuration,
		OpsPerSecReal: float64(totalOps) / globalDuration.Seconds(),
		OpsPerSecPure: pureThroughput,
		AvgLatency:    avgLatency,
		BytesPerOp:    bytes,
		AllocsPerOp:   allocs,
	}
}
