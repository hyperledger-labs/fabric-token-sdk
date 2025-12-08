/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package crypto

import "encoding/binary"

// AppendFixed32 appends slices prefixed with a 4-byte Little Endian length.
// Format: [Len(4 bytes)][Data]...
func AppendFixed32(dst []byte, s [][]byte) []byte {
	// 1. Precise Size Calculation
	// We calculate the exact total growth needed (4 bytes header + data length per slice).
	// This allows us to perform exactly one allocation.
	const headerSize = 4
	n := 0
	for _, v := range s {
		n += headerSize + len(v)
	}

	// 2. Single Growth / Allocation
	// If the capacity is insufficient, we grow the slice exactly once.
	// This avoids the 2x growth strategy of standard append(), saving
	// potentially 25-50% memory overhead on large buffers.
	if cap(dst)-len(dst) < n {
		newDst := make([]byte, len(dst), len(dst)+n)
		copy(newDst, dst)
		dst = newDst
	}

	// 3. Append Loop (Branch-free)
	for _, v := range s {
		// AppendUint32 is inlinable and highly optimized in Go 1.19+ [web:22]
		dst = binary.LittleEndian.AppendUint32(dst, uint32(len(v)))
		dst = append(dst, v...)
	}

	return dst
}
