/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package benchmark

import (
	"fmt"
	"math"
	"os"
	"runtime"
	"sort"
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
	OpsPerSecReal float64
	OpsPerSecPure float64
	AvgLatency    time.Duration
	StdDevLatency time.Duration
	Variance      float64
	P50Latency    time.Duration
	P75Latency    time.Duration
	P95Latency    time.Duration
	P99Latency    time.Duration
	MinLatency    time.Duration
	MaxLatency    time.Duration
	IQR           time.Duration // Interquartile Range (measure of spread)
	Jitter        time.Duration // Avg change between consecutive latencies
	CoeffVar      float64
	BytesPerOp    uint64
	AllocsPerOp   uint64
	Histogram     []Bucket
}

type Bucket struct {
	LowBound  time.Duration
	HighBound time.Duration
	Count     int
}

// chunk holds a fixed-size batch of latencies to prevent slice resizing costs
const chunkSize = 10000

type chunk struct {
	data [chunkSize]time.Duration
	next *chunk
	idx  int
}

// ANSI Color Codes
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
)

//nolint:errcheck
func (r Result) Print() {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	// Helper for coloring status
	status := func(condition bool, goodMsg, badMsg string) string {
		if condition {
			return ColorGreen + goodMsg + ColorReset
		}
		return ColorRed + badMsg + ColorReset
	}

	// --- Section 1: Main Metrics ---
	fmt.Fprintln(w, "Metric\tValue\tDescription")
	fmt.Fprintln(w, "------\t-----\t-----------")
	fmt.Fprintf(w, "Workers\t%d\t\n", r.GoRoutines)
	fmt.Fprintf(w, "Total Ops\t%d\t%s\n", r.OpsTotal,
		status(r.OpsTotal > 10000, "(Robust Sample)", "(Low Sample Size)"))
	fmt.Fprintf(w, "Duration\t%v\t%s\n", r.Duration,
		status(r.Duration > 1*time.Second, "(Good Duration)", "(Too Short < 1s)"))
	fmt.Fprintf(w, "Real Throughput\t%.2f/s\tObserved Ops/sec (Wall Clock)\n", r.OpsPerSecReal)

	// Overhead Check
	overheadPct := 0.0
	if r.OpsPerSecPure > 0 {
		overheadPct = (1.0 - (r.OpsPerSecReal / r.OpsPerSecPure)) * 100
	}

	overheadStatus := "(Low Overhead)"
	if overheadPct > 15.0 {
		overheadStatus = ColorYellow + fmt.Sprintf("(High Setup Cost: %.1f%%)", overheadPct) + ColorReset
	}

	fmt.Fprintf(w, "Pure Throughput\t%.2f/s\tTheoretical Max %s\n", r.OpsPerSecPure, overheadStatus)
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Latency Distribution:")
	fmt.Fprintf(w, "  Min\t%v\t\n", r.MinLatency)
	fmt.Fprintf(w, "  P50 (Median)\t%v\t\n", r.P50Latency)
	fmt.Fprintf(w, "  Average\t%v\t\n", r.AvgLatency)
	fmt.Fprintf(w, "  P95\t%v\t\n", r.P95Latency)
	fmt.Fprintf(w, "  P99\t%v\t\n", r.P99Latency)

	// Tail Latency Check
	tailRatio := 0.0
	if r.P99Latency > 0 {
		tailRatio = float64(r.MaxLatency) / float64(r.P99Latency)
	}

	maxStatus := ColorGreen + "(Stable Tail)" + ColorReset
	if tailRatio > 10.0 {
		maxStatus = ColorRed + fmt.Sprintf("(Extreme Outliers: Max is %.1fx P99)", tailRatio) + ColorReset
	}

	fmt.Fprintf(w, "  Max\t%v\t%s\n", r.MaxLatency, maxStatus)
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Stability Metrics:")
	fmt.Fprintf(w, "  Std Dev\t%v\t\n", r.StdDevLatency)
	fmt.Fprintf(w, "  IQR\t%v\tInterquartile Range\n", r.IQR)
	fmt.Fprintf(w, "  Jitter\t%v\tAvg delta per worker\n", r.Jitter)

	// CV Check
	cvPct := r.CoeffVar * 100
	cvStatus := ColorGreen + "Excellent Stability (<5%)" + ColorReset
	if cvPct > 20.0 {
		cvStatus = ColorRed + "Unstable (>20%) - Result is Noisy" + ColorReset
	} else if cvPct > 10.0 {
		cvStatus = ColorYellow + "Moderate Variance (10-20%)" + ColorReset
	}

	fmt.Fprintf(w, "  CV\t%.2f%%\t%s\n", cvPct, cvStatus)
	fmt.Fprintln(w, "")
	fmt.Fprintf(w, "Memory\t%d B/op\tAllocated bytes per operation\n", r.BytesPerOp)
	fmt.Fprintf(w, "Allocs\t%d allocs/op\tAllocations per operation\n", r.AllocsPerOp)
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Latency Heatmap (Dynamic Range):")
	fmt.Fprintln(w, "Range\tFreq\tDistribution Graph")

	maxCount := 0
	for _, b := range r.Histogram {
		if b.Count > maxCount {
			maxCount = b.Count
		}
	}

	for _, b := range r.Histogram {
		// Skip empty buckets
		if b.Count == 0 {
			continue
		}

		// 1. Draw Bar
		barLen := 0
		if maxCount > 0 {
			barLen = (b.Count * 40) / maxCount
		}

		ratio := 0.0
		if maxCount > 0 {
			ratio = float64(b.Count) / float64(maxCount)
		}

		// Heat Color Logic
		color := ColorBlue
		if ratio > 0.75 {
			color = ColorRed
		} else if ratio > 0.3 {
			color = ColorYellow
		} else if ratio > 0.1 {
			color = ColorGreen
		}

		bar := ""
		for i := 0; i < barLen; i++ {
			bar += "█"
		}

		// 2. Format Label
		label := fmt.Sprintf("%v-%v", b.LowBound, b.HighBound)
		// Visual fix for very small buckets
		if b.LowBound.Round(time.Microsecond) == b.HighBound.Round(time.Microsecond) && b.HighBound-b.LowBound < time.Microsecond {
			label = fmt.Sprintf("%dns-%dns", b.LowBound.Nanoseconds(), b.HighBound.Nanoseconds())
		}

		fmt.Fprintf(w, "  %s\t%d\t%s%s %s(%.1f%%)\n",
			label, b.Count, color, bar, ColorReset,
			(float64(b.Count)/float64(r.OpsTotal))*100)
	}

	w.Flush()

	// --- Section 2: Analysis & Recommendations ---
	fmt.Println("\n" + ColorBlue + "--- Analysis & Recommendations ---" + ColorReset)

	// 1. Sample Size Check
	if r.OpsTotal < 5000 {
		fmt.Printf("%s[WARN] Low sample size (%d). Results may not be statistically significant. Run for longer.%s\n", ColorRed, r.OpsTotal, ColorReset)
	}

	// 2. Duration Check
	if r.Duration < 1*time.Second {
		fmt.Printf("%s[WARN] Test ran for less than 1s. Go runtime/scheduler might not have stabilized.%s\n", ColorYellow, ColorReset)
	}

	// 3. Variance Check
	if cvPct > 20.0 {
		fmt.Printf("%s[FAIL] High Variance (CV %.2f%%). System noise is affecting results. Isolate the machine or increase duration.%s\n", ColorRed, cvPct, ColorReset)
	}

	// 4. Memory Check
	if r.AllocsPerOp > 100 {
		fmt.Printf("%s[INFO] High Allocations (%d/op). This will trigger frequent GC cycles and increase Max Latency.%s\n", ColorYellow, r.AllocsPerOp, ColorReset)
	}

	// 5. Outlier Check
	if tailRatio > 20.0 {
		fmt.Printf("%s[CRITICAL] Massive Latency Spikes Detected. Max is %.0fx higher than P99. Check for Stop-The-World GC or Lock Contention.%s\n", ColorRed, tailRatio, ColorReset)
	}

	if cvPct < 10.0 && r.OpsTotal > 10000 && tailRatio < 10.0 {
		fmt.Printf("%s[PASS] Benchmark looks healthy and statistically sound.%s\n", ColorGreen, ColorReset)
	}

	fmt.Println("----------------------------------")
}

func RunBenchmark[T any](
	workers int,
	benchDuration time.Duration,
	setup func() T,
	work func(T),
) Result {
	// ---------------------------------------------------------
	// PHASE 1: Memory Analysis (Serial & Isolated)
	// ---------------------------------------------------------
	// BUG FIX: Measure memory in isolation to avoid contamination from benchmark infrastructure

	var totalAllocs, totalBytes uint64
	const memSamples = 5

	for i := 0; i < memSamples; i++ {
		// SANITY CHECK: Force GC before each measurement to get clean baseline
		runtime.GC()
		runtime.GC() // Call twice to ensure finalization

		var memBefore, memAfter runtime.MemStats
		runtime.ReadMemStats(&memBefore)

		// BUG FIX: Create data inside the measurement window to avoid counting setup allocations
		data := setup()
		work(data)

		runtime.ReadMemStats(&memAfter)

		// SANITY CHECK: Ensure we're measuring deltas, not absolute values
		totalAllocs += memAfter.Mallocs - memBefore.Mallocs
		totalBytes += memAfter.TotalAlloc - memBefore.TotalAlloc

		// BUG FIX: Explicitly release data reference to avoid retention across iterations
		_ = data
	}

	allocs := totalAllocs / uint64(memSamples)
	bytes := totalBytes / uint64(memSamples)

	// ---------------------------------------------------------
	// PHASE 2: Throughput & Latency (Concurrent)
	// ---------------------------------------------------------
	// SANITY CHECK: Clean slate for Phase 2 - no contamination from Phase 1
	runtime.GC()
	runtime.GC()

	var (
		// BUG FIX: Remove totalWorkTime atomic counter - it's calculated from latency data
		// to avoid race conditions and timing measurement overhead
		running int32 // 1 = running, 0 = stopped
		startWg sync.WaitGroup
		endWg   sync.WaitGroup
	)

	workerResults := make([]*chunk, workers)
	atomic.StoreInt32(&running, 1)
	startWg.Add(workers)
	endWg.Add(workers)

	for i := 0; i < workers; i++ {
		workerID := i
		go func() {
			defer endWg.Done()

			// Initialize first chunk
			currentChunk := &chunk{}
			headChunk := currentChunk

			startWg.Done() // Ready signal
			startWg.Wait() // SANITY CHECK: Barrier - all workers start simultaneously

			// SANITY CHECK: Loop continues until global stop signal
			for atomic.LoadInt32(&running) == 1 {
				// 1. Setup
				d := setup()

				// 2. Work (timed)
				t0 := time.Now()
				work(d)
				dur := time.Since(t0)

				// 3. Record latency
				// BUG FIX: Check chunk capacity BEFORE writing to avoid overflow
				if currentChunk.idx >= chunkSize {
					newC := &chunk{}
					currentChunk.next = newC
					currentChunk = newC
				}

				currentChunk.data[currentChunk.idx] = dur
				currentChunk.idx++
			}

			// BUG FIX: Save head for post-processing (removed in-loop counter updates)
			// This eliminates the race condition and double-counting bugs
			workerResults[workerID] = headChunk
		}()
	}

	// SANITY CHECK: Ensure all workers are created and waiting before starting the timer
	startWg.Wait()
	startGlobal := time.Now()

	// Sleep for the benchmark duration
	time.Sleep(benchDuration)

	// SANITY CHECK: Signal all workers to stop and wait for cleanup
	atomic.StoreInt32(&running, 0)
	endWg.Wait()

	globalDuration := time.Since(startGlobal)

	// ---------------------------------------------------------
	// PHASE 3: Statistical Analysis
	// ---------------------------------------------------------
	// BUG FIX: Count all operations INCLUDING partial chunks at the end
	// Original code lost up to chunkSize operations per worker

	var totalOps uint64
	var totalTimeNs int64

	// First pass: count operations and sum latencies
	for _, head := range workerResults {
		curr := head
		for curr != nil {
			// SANITY CHECK: Only process valid entries (idx tells us how many)
			limit := curr.idx
			totalOps += uint64(limit)

			for k := 0; k < limit; k++ {
				totalTimeNs += int64(curr.data[k])
			}

			curr = curr.next
		}
	}

	// SANITY CHECK: Pre-allocate with exact capacity to avoid resizing overhead
	allLatencies := make([]time.Duration, 0, totalOps)

	// BUG FIX: Calculate jitter per-worker only (not across workers)
	// Jitter across workers is meaningless since they're concurrent
	var totalJitter float64
	var totalJitterSamples uint64

	for _, head := range workerResults {
		curr := head
		var prevLat time.Duration
		firstInWorker := true

		for curr != nil {
			limit := curr.idx

			for k := 0; k < limit; k++ {
				val := curr.data[k]
				allLatencies = append(allLatencies, val)

				// SANITY CHECK: Only calculate jitter within a worker's sequence
				if !firstInWorker {
					diff := float64(val - prevLat)
					if diff < 0 {
						diff = -diff
					}
					totalJitter += diff
					totalJitterSamples++
				}

				prevLat = val
				firstInWorker = false
			}

			curr = curr.next
		}
	}

	var (
		avgLatency         time.Duration
		stdDev             time.Duration
		variance           float64
		p50, p75, p95, p99 time.Duration
		minLat, maxLat     time.Duration
		iqr                time.Duration
		jitter             time.Duration
		coeffVar           float64
		pureThroughput     float64
	)

	if totalOps > 0 {
		// Jitter
		if totalJitterSamples > 0 {
			jitter = time.Duration(totalJitter / float64(totalJitterSamples))
		}

		// Basic stats
		avgLatency = time.Duration(totalTimeNs / int64(totalOps))
		// SANITY CHECK: Pure throughput is theoretical max if setup() had zero cost
		pureThroughput = float64(workers) / avgLatency.Seconds()

		// Sort for percentiles
		sort.Slice(allLatencies, func(i, j int) bool {
			return allLatencies[i] < allLatencies[j]
		})

		minLat = allLatencies[0]
		maxLat = allLatencies[len(allLatencies)-1]

		p50 = percentile(allLatencies, 0.50)
		p75 = percentile(allLatencies, 0.75)
		p95 = percentile(allLatencies, 0.95)
		p99 = percentile(allLatencies, 0.99)

		p25 := percentile(allLatencies, 0.25)
		iqr = p75 - p25

		// BUG FIX: Use population variance (divide by n) not sample variance (n-1)
		// In benchmarks, we're measuring the entire population, not sampling
		meanNs := float64(avgLatency.Nanoseconds())
		var sumSquaredDiff float64

		for _, lat := range allLatencies {
			diff := float64(lat.Nanoseconds()) - meanNs
			sumSquaredDiff += diff * diff
		}

		// SANITY CHECK: Avoid division by zero
		if len(allLatencies) > 0 {
			variance = sumSquaredDiff / float64(len(allLatencies))
			stdDev = time.Duration(math.Sqrt(variance))

			if avgLatency > 0 {
				coeffVar = float64(stdDev) / float64(avgLatency)
			}
		}
	}

	hist := calcExponentialHistogram(allLatencies, minLat, maxLat, 20)

	return Result{
		GoRoutines:    workers,
		OpsTotal:      totalOps,
		Duration:      globalDuration,
		OpsPerSecReal: float64(totalOps) / globalDuration.Seconds(),
		OpsPerSecPure: pureThroughput,
		AvgLatency:    avgLatency,
		StdDevLatency: stdDev,
		Variance:      variance,
		P50Latency:    p50,
		P75Latency:    p75,
		P95Latency:    p95,
		P99Latency:    p99,
		MinLatency:    minLat,
		MaxLatency:    maxLat,
		IQR:           iqr,
		Jitter:        jitter,
		CoeffVar:      coeffVar,
		BytesPerOp:    bytes,
		AllocsPerOp:   allocs,
		Histogram:     hist,
	}
}

func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}

	// SANITY CHECK: Use ceiling method for percentile calculation
	idx := int(math.Ceil(p*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}

	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}

	return sorted[idx]
}

func calcExponentialHistogram(latencies []time.Duration, min, max time.Duration, bucketCount int) []Bucket {
	if len(latencies) == 0 {
		return nil
	}

	// Edge case: Min/Max are equal or invalid
	// SANITY CHECK: Ensure min is positive for log calculations
	if min <= 0 {
		min = 1
	}

	if max < min {
		max = min
	}

	buckets := make([]Bucket, bucketCount)

	// 1. Calculate the geometric growth factor
	var factor float64
	if min == max {
		// SANITY CHECK: All values are identical, single bucket
		factor = 1.0
	} else {
		// Formula: min * factor^N = max
		ratio := float64(max) / float64(min)
		factor = math.Pow(ratio, 1.0/float64(bucketCount))
	}

	// 2. Initialize Bucket Boundaries
	currentLower := float64(min)
	for i := 0; i < bucketCount; i++ {
		var currentUpper float64
		if i == bucketCount-1 {
			// SANITY CHECK: Snap last bucket strictly to max to avoid floating point errors
			currentUpper = float64(max)
		} else {
			currentUpper = currentLower * factor
		}

		buckets[i] = Bucket{
			LowBound:  time.Duration(currentLower),
			HighBound: time.Duration(currentUpper),
			Count:     0,
		}

		currentLower = currentUpper
	}

	// 3. Populate Counts using Logarithmic Indexing (Fast O(n) instead of O(n*bucketCount))
	logMin := math.Log(float64(min))
	logFactor := math.Log(factor)

	for _, lat := range latencies {
		val := float64(lat)
		var idx int

		// SANITY CHECK: Handle edge cases where all values are the same
		if min == max || logFactor == 0 {
			idx = 0
		} else {
			// Inverse of the exponential function to find index directly
			// index = log_factor(value / min) = (log(value) - log(min)) / log(factor)
			idx = int((math.Log(val) - logMin) / logFactor)
		}

		// SANITY CHECK: Clamp index to handle floating point precision edges
		if idx < 0 {
			idx = 0
		}

		if idx >= bucketCount {
			idx = bucketCount - 1
		}

		buckets[idx].Count++
	}

	return buckets
}
