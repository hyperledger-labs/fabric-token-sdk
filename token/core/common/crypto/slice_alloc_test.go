/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package crypto

import (
	"bytes"
	"encoding/binary"
	"runtime"
	"testing"
)

// helper to create test slices
func makeSlices(num, size int) [][]byte {
	s := make([][]byte, num)
	for i := range num {
		b := make([]byte, size)
		for j := range b {
			b[j] = byte((i + j) % 256)
		}
		s[i] = b
	}

	return s
}

// avgAllocBytes runs fn 'runs' times and returns the average number of bytes
// allocated per run using runtime.TotalAlloc deltas.
func avgAllocBytes(fn func(), runs int) uint64 {
	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)
	for range runs {
		fn()
	}
	runtime.GC()
	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	// TotalAlloc is cumulative; take the delta and divide by runs
	if after.TotalAlloc < before.TotalAlloc {
		return 0
	}

	return (after.TotalAlloc - before.TotalAlloc) / uint64(runs) // #nosec G115
}

func TestAppendFixed32VsBytesJoinAllocations(t *testing.T) {
	// Configurable parameters: adjust if you want heavier or lighter workloads
	numSlices := 100
	sliceSize := 1024 // 1KB each
	runs := 50

	s := makeSlices(numSlices, sliceSize)

	// Measure allocation counts
	allocsAppend := testing.AllocsPerRun(runs, func() {
		var dst []byte
		dst = AppendFixed32(dst, s)
		_ = dst
	})

	allocsJoin := testing.AllocsPerRun(runs, func() {
		pieces := make([][]byte, 0, len(s)*2)
		for _, v := range s {
			hdr := make([]byte, 4)
			binary.LittleEndian.PutUint32(hdr, uint32(len(v))) // #nosec G115
			pieces = append(pieces, hdr, v)
		}
		dst := bytes.Join(pieces, nil)
		_ = dst
	})

	// Measure average bytes allocated per run
	avgBytesAppend := avgAllocBytes(func() {
		var dst []byte
		dst = AppendFixed32(dst, s)
		_ = dst
	}, 10)

	avgBytesJoin := avgAllocBytes(func() {
		pieces := make([][]byte, 0, len(s)*2)
		for _, v := range s {
			hdr := make([]byte, 4)
			binary.LittleEndian.PutUint32(hdr, uint32(len(v))) // #nosec G115
			pieces = append(pieces, hdr, v)
		}
		dst := bytes.Join(pieces, nil)
		_ = dst
	}, 10)

	t.Logf("AppendFixed32: allocs/run=%v, avgBytes/run=%d", allocsAppend, avgBytesAppend)
	t.Logf("bytes.Join:   allocs/run=%v, avgBytes/run=%d", allocsJoin, avgBytesJoin)

	// Expect AppendFixed32 to allocate no more than bytes.Join in terms of allocation count.
	if allocsAppend > allocsJoin {
		t.Fatalf("unexpected: AppendFixed32 allocs (%v) > bytes.Join allocs (%v)", allocsAppend, allocsJoin)
	}
}
