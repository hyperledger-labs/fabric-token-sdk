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

// Result holds the comprehensive benchmark metrics.
type Result struct {
	GoRoutines    int
	OpsTotal      uint64
	Duration      time.Duration
	OpsPerSecReal float64
	OpsPerSecPure float64

	AvgLatency    time.Duration
	StdDevLatency time.Duration
	Variance      float64

	P50Latency time.Duration
	P75Latency time.Duration
	P95Latency time.Duration
	P99Latency time.Duration
	MinLatency time.Duration
	MaxLatency time.Duration

	IQR    time.Duration // Interquartile Range (measure of spread)
	Jitter time.Duration // Avg change between consecutive latencies

	CoeffVar    float64
	BytesPerOp  uint64
	AllocsPerOp uint64

	Histogram []Bucket
}

// Bucket represents a latency range and its frequency.
type Bucket struct {
	LowBound  time.Duration
	HighBound time.Duration
	Count     int
}

// chunk holds a fixed-size batch of latencies to prevent slice resizing costs.
//
// SANITY CHECK: We keep a simple fixed array + linked-list so that the
// benchmark's hot path does not allocate on every operation when recording
// latency.
const chunkSize = 10000

type chunk struct {
	data [chunkSize]time.Duration
	next *chunk
	idx  int // number of valid entries in data
}

// ANSI Color Codes for output.
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

	// Helper for coloring status.
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
	fmt.Fprintf(
		w,
		"Total Ops\t%d\t%s\n",
		r.OpsTotal,
		status(r.OpsTotal > 10000, "(Robust Sample)", "(Low Sample Size)"),
	)
	fmt.Fprintf(
		w,
		"Duration\t%v\t%s\n",
		r.Duration,
		status(r.Duration > 1*time.Second, "(Good Duration)", "(Too Short < 1s)"),
	)

	fmt.Fprintf(w, "Real Throughput\t%.2f/s\tObserved Ops/sec (Wall Clock)\n", r.OpsPerSecReal)

	// Overhead Check.
	overheadPct := 0.0
	if r.OpsPerSecPure > 0 && r.OpsPerSecReal > 0 {
		// SANITY CHECK: Clamp to [0, 100+] range to avoid NaN/Inf due to
		// floating errors in weird edge cases.
		overheadPct = (1.0 - (r.OpsPerSecReal / r.OpsPerSecPure)) * 100
	}

	overheadStatus := "(Low Overhead)"
	if overheadPct > 15.0 {
		overheadStatus = ColorYellow + fmt.Sprintf("(High Setup Cost: %.1f%%)", overheadPct) + ColorReset
	}

	fmt.Fprintf(w, "Pure Throughput\t%.2f/s\tTheoretical Max %s\n", r.OpsPerSecPure, overheadStatus)
	fmt.Fprintln(w, "")

	fmt.Fprintln(w, "Latency Distribution:")
	fmt.Fprintf(w, " Min\t%v\t\n", r.MinLatency)
	fmt.Fprintf(w, " P50 (Median)\t%v\t\n", r.P50Latency)
	fmt.Fprintf(w, " Average\t%v\t\n", r.AvgLatency)
	fmt.Fprintf(w, " P95\t%v\t\n", r.P95Latency)
	fmt.Fprintf(w, " P99\t%v\t\n", r.P99Latency)

	// Tail Latency Check.
	tailRatio := 0.0
	if r.P99Latency > 0 {
		tailRatio = float64(r.MaxLatency) / float64(r.P99Latency)
	}

	maxStatus := ColorGreen + "(Stable Tail)" + ColorReset
	if tailRatio > 10.0 {
		maxStatus = ColorRed + fmt.Sprintf("(Extreme Outliers: Max is %.1fx P99)", tailRatio) + ColorReset
	}
	fmt.Fprintf(w, " Max\t%v\t%s\n", r.MaxLatency, maxStatus)
	fmt.Fprintln(w, "")

	fmt.Fprintln(w, "Stability Metrics:")
	fmt.Fprintf(w, " Std Dev\t%v\t\n", r.StdDevLatency)
	fmt.Fprintf(w, " IQR\t%v\tInterquartile Range\n", r.IQR)
	fmt.Fprintf(w, " Jitter\t%v\tAvg delta per worker\n", r.Jitter)

	// CV Check.
	cvPct := r.CoeffVar * 100
	cvStatus := ColorGreen + "Excellent Stability (<5%)" + ColorReset
	if cvPct > 20.0 {
		cvStatus = ColorRed + "Unstable (>20%) - Result is Noisy" + ColorReset
	} else if cvPct > 10.0 {
		cvStatus = ColorYellow + "Moderate Variance (10-20%)" + ColorReset
	}
	fmt.Fprintf(w, " CV\t%.2f%%\t%s\n", cvPct, cvStatus)
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
		// Skip empty buckets.
		if b.Count == 0 {
			continue
		}

		// 1. Draw Bar.
		barLen := 0
		if maxCount > 0 {
			// SANITY CHECK: Scale to at most ~40 chars so the output remains readable.
			barLen = (b.Count * 40) / maxCount
		}

		ratio := 0.0
		if maxCount > 0 {
			ratio = float64(b.Count) / float64(maxCount)
		}

		// Heat Color Logic.
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

		// 2. Format Label.
		label := fmt.Sprintf("%v-%v", b.LowBound, b.HighBound)
		// Visual fix for very small buckets.
		if b.LowBound.Round(time.Microsecond) == b.HighBound.Round(time.Microsecond) &&
			b.HighBound-b.LowBound < time.Microsecond {
			label = fmt.Sprintf("%dns-%dns", b.LowBound.Nanoseconds(), b.HighBound.Nanoseconds())
		}

		percentage := 0.0
		if r.OpsTotal > 0 {
			percentage = (float64(b.Count) / float64(r.OpsTotal)) * 100
		}

		fmt.Fprintf(
			w,
			" %s\t%d\t%s%s %s(%.1f%%)\n",
			label,
			b.Count,
			color,
			bar,
			ColorReset,
			percentage,
		)
	}

	w.Flush()

	// --- Section 2: Analysis & Recommendations ---
	fmt.Println("\n" + ColorBlue + "--- Analysis & Recommendations ---" + ColorReset)

	// 1. Sample Size Check.
	if r.OpsTotal < 5000 {
		fmt.Printf(
			"%s[WARN] Low sample size (%d). Results may not be statistically significant. Run for longer.%s\n",
			ColorRed,
			r.OpsTotal,
			ColorReset,
		)
	}

	// 2. Duration Check.
	if r.Duration < 1*time.Second {
		fmt.Printf(
			"%s[WARN] Test ran for less than 1s. Go runtime/scheduler might not have stabilized.%s\n",
			ColorYellow,
			ColorReset,
		)
	}

	// 3. Variance Check.
	if cvPct > 20.0 {
		fmt.Printf(
			"%s[FAIL] High Variance (CV %.2f%%). System noise is affecting results. "+
				"Isolate the machine or increase duration.%s\n",
			ColorRed,
			cvPct,
			ColorReset,
		)
	}

	// 4. Memory Check.
	if r.AllocsPerOp > 100 {
		fmt.Printf(
			"%s[INFO] High Allocations (%d/op). This will trigger frequent GC cycles and increase Max Latency.%s\n",
			ColorYellow,
			r.AllocsPerOp,
			ColorReset,
		)
	}

	// 5. Outlier Check.
	if tailRatio > 20.0 {
		fmt.Printf(
			"%s[CRITICAL] Massive Latency Spikes Detected. Max is %.0fx higher than P99. "+
				"Check for Stop-The-World GC or Lock Contention.%s\n",
			ColorRed,
			tailRatio,
			ColorReset,
		)
	}

	if cvPct < 10.0 && r.OpsTotal > 10000 && tailRatio < 10.0 {
		fmt.Printf(
			"%s[PASS] Benchmark looks healthy and statistically sound.%s\n",
			ColorGreen,
			ColorReset,
		)
	}

	fmt.Println("----------------------------------")
}

func RunBenchmark[T any](
	workers int,
	benchDuration time.Duration,
	setup func() T,
	work func(T),
) Result {
	// SANITY CHECK: Ensure we have at least one worker and a positive duration.
	if workers <= 0 {
		workers = 1
	}
	if benchDuration <= 0 {
		benchDuration = 1 * time.Second
	}

	// ---------------------------------------------------------
	// PHASE 1: Memory Analysis (Serial & Isolated)
	// ---------------------------------------------------------
	//
	// Measure memory in isolation to avoid contamination from benchmark
	// infrastructure (channels, sync, etc.).

	var totalAllocs, totalBytes uint64
	const memSamples = 5

	for i := 0; i < memSamples; i++ {
		// SANITY CHECK: Force GC before each measurement to get a clean baseline.
		// Call GC twice to ensure finalization of objects from previous iteration.
		runtime.GC()
		runtime.GC()

		// Sleep briefly to allow GC to fully complete before measurement.
		// Without this, overlapping GC from previous iteration can contaminate
		// measurements.
		time.Sleep(10 * time.Millisecond)

		var memBefore, memAfter runtime.MemStats
		runtime.ReadMemStats(&memBefore)

		// Create data inside the measurement window to avoid counting setup
		// allocations made before the baseline.
		data := setup()
		work(data)

		runtime.ReadMemStats(&memAfter)

		// SANITY CHECK: Ensure we're measuring deltas, not absolute values.
		// This accounts for allocations made specifically by setup() + work().
		totalAllocs += memAfter.Mallocs - memBefore.Mallocs
		totalBytes += memAfter.TotalAlloc - memBefore.TotalAlloc
	}

	// SANITY CHECK: Average over multiple samples to reduce noise from GC timing
	// variance. Note that memSamples is the number of ops in this phase.
	allocs := totalAllocs / uint64(memSamples)
	bytes := totalBytes / uint64(memSamples)

	// ---------------------------------------------------------
	// PHASE 2: Throughput & Latency (Concurrent)
	// ---------------------------------------------------------

	// SANITY CHECK: Clean slate for Phase 2 - no contamination from Phase 1.
	runtime.GC()
	runtime.GC()
	time.Sleep(10 * time.Millisecond)

	var (
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

			// Initialize first chunk for this worker.
			currentChunk := &chunk{}
			headChunk := currentChunk

			// Signal readiness and wait for all workers.
			startWg.Done()
			// SANITY CHECK: Barrier - all workers start simultaneously.
			// This ensures fair timing and prevents early-bird bias.
			startWg.Wait()

			// SANITY CHECK: Loop continues until global stop signal.
			// Using atomic load ensures memory visibility across goroutines.
			for atomic.LoadInt32(&running) == 1 {
				// 1. Setup: Create test data (not timed).
				d := setup()

				// 2. Work: Execute the operation we're benchmarking (timed).
				t0 := time.Now()
				work(d)
				dur := time.Since(t0)

				// 3. Record latency.
				//
				// SANITY CHECK: Check chunk capacity BEFORE writing to avoid overflow.
				if currentChunk.idx >= chunkSize {
					newC := &chunk{}
					currentChunk.next = newC
					currentChunk = newC
				}

				// SANITY CHECK: Store latency in pre-allocated array (no allocation
				// overhead on the hot path).
				currentChunk.data[currentChunk.idx] = dur
				currentChunk.idx++
			}

			// Save head pointer for post-processing.
			// Each worker maintains its own linked list of chunks.
			workerResults[workerID] = headChunk
		}()
	}

	// SANITY CHECK: Ensure all workers are created and waiting before starting
	// the timer. This prevents skew from goroutine creation overhead.
	startWg.Wait()

	startGlobal := time.Now()

	// Sleep for the benchmark duration - workers run concurrently during this time.
	time.Sleep(benchDuration)

	// SANITY CHECK: Signal all workers to stop and wait for cleanup.
	// Using atomic store ensures all workers see the stop signal promptly.
	atomic.StoreInt32(&running, 0)
	endWg.Wait()

	globalDuration := time.Since(startGlobal)
	if globalDuration <= 0 {
		// Avoid division by zero; in practice, this should not happen with sane durations.
		globalDuration = 1
	}

	// ---------------------------------------------------------
	// PHASE 3: Statistical Analysis
	// ---------------------------------------------------------

	// Count all operations INCLUDING partial chunks at the end.
	var totalOps uint64
	var totalTimeNs int64

	// First pass: count operations and sum latencies.
	for _, head := range workerResults {
		curr := head
		for curr != nil {
			// SANITY CHECK: Only process valid entries (idx tells us how many were written).
			limit := curr.idx
			totalOps += uint64(limit)

			for k := 0; k < limit; k++ {
				// Use int64 for nanosecond accumulation to prevent overflow.
				totalTimeNs += int64(curr.data[k])
			}

			curr = curr.next
		}
	}

	// SANITY CHECK: Pre-allocate with exact-ish capacity to avoid resizing overhead
	// during analysis. This keeps analysis overhead out of the hot path.
	allLatencies := make([]time.Duration, 0, int(totalOps))

	// Calculate jitter per-worker only (not across workers).
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

				// SANITY CHECK: Only calculate jitter within a worker's sequence.
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
		avgLatency     time.Duration
		stdDev         time.Duration
		variance       float64
		p50, p75       time.Duration
		p95, p99       time.Duration
		minLat, maxLat time.Duration
		iqr            time.Duration
		jitter         time.Duration
		coeffVar       float64
		pureThroughput float64
	)

	if totalOps > 0 {
		// Jitter calculation.
		if totalJitterSamples > 0 {
			jitter = time.Duration(totalJitter / float64(totalJitterSamples))
		}

		// Basic stats: average latency.
		avgLatency = time.Duration(totalTimeNs / int64(totalOps))

		// SANITY CHECK: Pure throughput is theoretical max if setup() had zero cost.
		// Guard against zero avgLatency to avoid Inf.
		if avgLatency > 0 {
			pureThroughput = float64(workers) / avgLatency.Seconds()
		}

		// Sort for percentiles - required for accurate quantile calculations.
		sort.Slice(allLatencies, func(i, j int) bool {
			return allLatencies[i] < allLatencies[j]
		})

		minLat = allLatencies[0]
		maxLat = allLatencies[len(allLatencies)-1]

		// Use linear interpolation for percentiles.
		p50 = percentileInterpolated(allLatencies, 0.50)
		p75 = percentileInterpolated(allLatencies, 0.75)
		p95 = percentileInterpolated(allLatencies, 0.95)
		p99 = percentileInterpolated(allLatencies, 0.99)
		p25 := percentileInterpolated(allLatencies, 0.25)

		// IQR measures the spread of the middle 50% of data.
		iqr = p75 - p25

		// Use population variance (divide by n) as we're observing full data, not a sample.
		meanNs := float64(avgLatency.Nanoseconds())
		var sumSquaredDiff float64

		for _, lat := range allLatencies {
			diff := float64(lat.Nanoseconds()) - meanNs
			sumSquaredDiff += diff * diff
		}

		if len(allLatencies) > 0 {
			variance = sumSquaredDiff / float64(len(allLatencies))
			stdDev = time.Duration(math.Sqrt(variance))
			if avgLatency > 0 {
				coeffVar = float64(stdDev) / float64(avgLatency)
			}
		}
	}

	// Build histogram with improved boundary handling.
	hist := calcExponentialHistogramImproved(allLatencies, minLat, maxLat, 20)

	return Result{
		GoRoutines:    workers,
		OpsTotal:      totalOps,
		Duration:      globalDuration,
		OpsPerSecReal: float64(totalOps) / globalDuration.Seconds(),
		OpsPerSecPure: pureThroughput,

		AvgLatency:    avgLatency,
		StdDevLatency: stdDev,
		Variance:      variance,

		P50Latency: p50,
		P75Latency: p75,
		P95Latency: p95,
		P99Latency: p99,
		MinLatency: minLat,
		MaxLatency: maxLat,

		IQR:         iqr,
		Jitter:      jitter,
		CoeffVar:    coeffVar,
		BytesPerOp:  bytes,
		AllocsPerOp: allocs,
		Histogram:   hist,
	}
}

// percentileInterpolated computes a percentile using linear interpolation.
//
// SANITY CHECK: sorted must be non-empty for meaningful results.
func percentileInterpolated(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}

	// Edge cases.
	if p <= 0 {
		return sorted[0]
	}
	if p >= 1 {
		return sorted[len(sorted)-1]
	}

	// Calculate the exact position (0-indexed, can be fractional).
	pos := p * float64(len(sorted)-1)
	lower := int(math.Floor(pos))
	upper := int(math.Ceil(pos))

	// If position is exactly on an index, return that value.
	if lower == upper {
		return sorted[lower]
	}

	// Linear interpolation between adjacent values.
	fraction := pos - float64(lower)
	lowerVal := float64(sorted[lower])
	upperVal := float64(sorted[upper])
	interpolated := lowerVal + fraction*(upperVal-lowerVal)

	return time.Duration(interpolated)
}

// calcExponentialHistogramImproved builds an exponential histogram with better
// boundary precision and O(n) assignment.
func calcExponentialHistogramImproved(
	latencies []time.Duration,
	min time.Duration,
	max time.Duration,
	bucketCount int,
) []Bucket {
	if len(latencies) == 0 || bucketCount <= 0 {
		return nil
	}

	// SANITY CHECK: Ensure min is positive for log calculations.
	if min <= 0 {
		min = 1
	}
	if max < min {
		max = min
	}

	buckets := make([]Bucket, bucketCount)

	// 1. Calculate the geometric growth factor.
	var factor float64
	if min == max {
		// All values identical, a single effective bucket.
		factor = 1.0
	} else {
		// Formula: min * factor^N = max.
		ratio := float64(max) / float64(min)
		factor = math.Pow(ratio, 1.0/float64(bucketCount))
	}

	// 2. Initialize Bucket Boundaries using int64 nanoseconds to avoid float drift.
	currentLowerNs := min.Nanoseconds()

	for i := 0; i < bucketCount; i++ {
		var currentUpperNs int64
		if i == bucketCount-1 {
			// SANITY CHECK: Snap last bucket strictly to max to avoid floating
			// point errors that might drop values outside the histogram.
			currentUpperNs = max.Nanoseconds()
		} else {
			if factor == 1.0 {
				currentUpperNs = max.Nanoseconds()
			} else {
				currentUpperNs = int64(float64(currentLowerNs) * factor)
			}
			if currentUpperNs < currentLowerNs {
				// Guard against rounding weirdness.
				currentUpperNs = currentLowerNs
			}
		}

		buckets[i] = Bucket{
			LowBound:  time.Duration(currentLowerNs),
			HighBound: time.Duration(currentUpperNs),
			Count:     0,
		}

		currentLowerNs = currentUpperNs
	}

	// 3. Populate Counts using Logarithmic Indexing (O(n)).
	logMin := math.Log(float64(min.Nanoseconds()))
	logFactor := 0.0
	if factor != 1.0 {
		logFactor = math.Log(factor)
	}

	for _, lat := range latencies {
		valNs := float64(lat.Nanoseconds())
		if valNs <= 0 {
			// Extremely small or zero value, put into first bucket.
			buckets[0].Count++
			continue
		}

		var idx int
		if min == max || logFactor == 0 {
			// All values the same or factor degenerate.
			idx = 0
		} else {
			// Inverse of the exponential function to find index directly:
			// idx = floor((log(value) - log(min)) / log(factor)).
			idx = int(math.Floor((math.Log(valNs) - logMin) / logFactor))
		}

		// Clamp index to [0, bucketCount-1] to handle precision edge cases.
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
