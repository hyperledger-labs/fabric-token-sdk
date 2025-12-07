# Go Pprof Memory Analyzer

A specialized command-line tool for deep-diving into Go heap profiles (`pprof`). 
This analyzer goes beyond standard pprof tools by applying heuristics to detect common memory anti-patterns, potential leaks, and high-pressure allocation hotspots.

## Features

This tool parses a standard `pprof` heap profile and generates a 7-section unified report:

1.  **Anti-Pattern Detection**: Automatically flags common mistakes (e.g., `time.After` in loops, heavy JSON reflection, unoptimized slice growth).
2.  **Hot Lines**: Pinpoints the exact file and line number responsible for the most allocations.
3.  **Business Logic Context**: aggregates data by pprof labels (if present) to show cost per request/worker/context.
4.  **Top Object Producers**: Identifies functions creating the most garbage (GC pressure), calculated by allocation count and average object size.
5.  **Leak Candidates**: Highlights functions with high "In-Use" memory but low "Allocated" throughput—a strong signal for memory leaks or unbounded caches.
6.  **Root Cause Traces**: Displays the call stack for the top 5 heavy allocators to show *who* is calling the expensive function.
7.  **ASCII Flame Graph**: Visualizes the call tree directly in your terminal for paths consuming >1% of memory.

## Installation

You can run the tool directly or build it as a binary.

### Build
```
make memcheck
```

### Run
```
memcheck <path-to-heap-profile.pb.gz>
```

Or run directly with `go run`:
```
go run main.go heap.pb.gz
```

*Note: You must provide a standard Go heap profile (proto format). You can generate one from a running Go app using:*
`curl -o heap.pb.gz http://localhost:6060/debug/pprof/heap`

## Report Sections Explained

### 1. Detected Anti-Patterns & Heuristics
Checks your profile against a list of known Go performance pitfalls:
- **Loop Timer Leak**: Misuse of `time.After` inside loops.
- **Repeated RegEx**: `regexp.Compile` appearing in hot paths.
- **Heavy JSON**: Excessive allocation in `json.Unmarshal`.
- **Interface Boxing**: High cost of converting concrete types to `interface{}`.
- **Slice/Map Growth**: High allocations in `growslice` or `mapassign` (suggests missing capacity pre-allocation).

### 2. Hot Lines
Shows the exact source code location (File:Line) for the top allocators. This is often more useful than function names alone.

### 3. Business Logic Context (Labels)
If your application uses pprof labels (e.g., `pprof.Do`), this section breaks down memory usage by those labels (e.g., per HTTP route or background worker ID).

### 4. Top Object Producers
Focuses on **GC Pressure**. Functions listed here churn through many small objects, causing the Garbage Collector to run frequently, even if total memory usage is low.

### 5. Persistent Memory (Leak Candidates)
Focuses on **RAM Usage**. Lists functions that allocate memory that *stays* allocated.
- **High In-Use %**: Suggests a cache, global variable, or memory leak.
- **Diagnosis**: The tool provides specific advice (e.g., "Unbounded Map?", "Check capacity reset") based on the ratio of In-Use vs. Allocated bytes.

### 6. Root Cause Trace
For the top 5 allocators, this prints the stack trace up to the root. It attempts to tag the "Likely Cause" by skipping standard library functions to find your code.

### 7. ASCII Flame Graph
A text-based tree view of memory consumption.
- `├──` indicates a child call.
- Shows total bytes and percentage of heap for that path.
- Hides paths contributing <1% to reduce noise.
