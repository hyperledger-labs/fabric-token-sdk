/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nfttx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateBytesUUID(t *testing.T) {
	uuid := GenerateBytesUUID()

	// UUID should be 16 bytes
	assert.Len(t, uuid, 16)

	// Check variant bits (10xx xxxx)
	assert.Equal(t, byte(0x80), uuid[8]&0xc0, "variant bits should be 10")

	// Check version bits (0100 xxxx for version 4)
	assert.Equal(t, byte(0x40), uuid[6]&0xf0, "version bits should be 0100")
}

func TestGenerateUUID(t *testing.T) {
	uuid := GenerateUUID()

	// UUID string format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	// Length should be 36 characters (32 hex + 4 hyphens)
	assert.Len(t, uuid, 36)

	// Check format with hyphens at correct positions
	assert.Equal(t, "-", string(uuid[8]))
	assert.Equal(t, "-", string(uuid[13]))
	assert.Equal(t, "-", string(uuid[18]))
	assert.Equal(t, "-", string(uuid[23]))
}

func TestGenerateUUID_Uniqueness(t *testing.T) {
	// Generate multiple UUIDs and ensure they're unique
	uuids := make(map[string]bool)
	iterations := 1000

	for range iterations {
		uuid := GenerateUUID()
		require.False(t, uuids[uuid], "UUID collision detected: %s", uuid)
		uuids[uuid] = true
	}

	assert.Len(t, uuids, iterations, "should have generated %d unique UUIDs", iterations)
}

func TestIdBytesToStr(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "all zeros",
			input:    make([]byte, 16),
			expected: "00000000-0000-0000-0000-000000000000",
		},
		{
			name:     "all ones",
			input:    []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
			expected: "ffffffff-ffff-ffff-ffff-ffffffffffff",
		},
		{
			name:     "mixed values",
			input:    []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef},
			expected: "01234567-89ab-cdef-0123-456789abcdef",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := idBytesToStr(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Made with Bob
