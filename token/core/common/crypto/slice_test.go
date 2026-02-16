/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package crypto

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Unit Test: Verifies correct Little Endian encoding and data integrity
func TestAppendFixed32(t *testing.T) {
	tests := []struct {
		name     string
		input    [][]byte
		expected []byte
	}{
		{
			name: "Basic Join",
			input: [][]byte{
				[]byte("Go"),
				[]byte("Lang"),
			},
			// Expect: [Len:2][G][o] [Len:4][L][a][n][g]
			// Little Endian 2: 0x02, 0x00, 0x00, 0x00
			expected: []byte{
				0x02, 0x00, 0x00, 0x00, 'G', 'o',
				0x04, 0x00, 0x00, 0x00, 'L', 'a', 'n', 'g',
			},
		},
		{
			name:     "Empty Input",
			input:    [][]byte{},
			expected: nil, // Or empty slice depending on init
		},
		{
			name: "Contains Empty Slice",
			input: [][]byte{
				[]byte("Hi"),
				{},
			},
			// Expect: [Len:2][H][i] [Len:0]
			expected: []byte{
				0x02, 0x00, 0x00, 0x00, 'H', 'i',
				0x00, 0x00, 0x00, 0x00,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Using nil as dst to force new allocation
			result := AppendFixed32(nil, tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Verification Test: Reusing an existing buffer
func TestAppendFixed32_ReuseBuffer(t *testing.T) {
	buffer := make([]byte, 0, 1024)
	buffer = append(buffer, 0xFF) // Simulating existing data (dirty buffer)

	input := [][]byte{[]byte("A")}
	result := AppendFixed32(buffer, input)

	// Check that we didn't lose the prefix 0xFF
	assert.Equal(t, byte(0xFF), result[0])
	// Check the new data starts at index 1
	// Len: 1 (0x01 00 00 00) + 'A'
	expectedPayload := []byte{0x01, 0x00, 0x00, 0x00, 'A'}
	assert.Equal(t, expectedPayload, result[1:])
}

// --- Benchmarks ---

// Setup helper for benchmarks
func generateBenchmarkData(count, size int) [][]byte {
	data := make([][]byte, count)
	for i := range count {
		data[i] = bytes.Repeat([]byte{'a'}, size)
	}
	return data
}

// Optimized Approach
func BenchmarkAppendFixed32(b *testing.B) {
	// Scenario: 1000 items, 256 bytes each (~250KB total)
	data := generateBenchmarkData(1000, 256)

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		// Use nil to strictly measure allocation of the result
		_ = AppendFixed32(nil, data)
	}
}

// Comparison: Naive Loop (No pre-calculation)
func BenchmarkAppendFixed32_Naive(b *testing.B) {
	data := generateBenchmarkData(1000, 256)

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		var dst []byte
		for _, v := range data {
			dst = binary.LittleEndian.AppendUint32(dst, uint32(len(v))) // #nosec G115
			dst = append(dst, v...)
		}
	}
}
