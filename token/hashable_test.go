/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"hash"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHashable_Raw_PanicOnPartialWrite tests panic when hash.Write returns fewer bytes than expected.
func TestHashable_Raw_PanicOnPartialWrite(t *testing.T) {
	data := []byte("test data")

	// Save original hashFunc and restore after test
	originalHashFunc := hashFunc
	defer func() {
		hashFunc = originalHashFunc
	}()

	// Override hashFunc to return a mock that writes fewer bytes
	hashFunc = func() hash.Hash {
		mock := &HashMock{}
		mock.WriteReturns(len(data)-1, nil) // Return fewer bytes than expected
		mock.SumReturns(make([]byte, 32))
		mock.SizeReturns(32)
		mock.BlockSizeReturns(64)
		return mock
	}

	h := Hashable(data)
	assert.Panics(t, func() {
		h.Raw()
	}, "Should panic when hash.Write returns fewer bytes than expected")
}

// TestHashable_Raw_PanicOnWriteError tests panic when hash.Write returns an error.
func TestHashable_Raw_PanicOnWriteError(t *testing.T) {
	data := []byte("test data")

	// Save original hashFunc and restore after test
	originalHashFunc := hashFunc
	defer func() {
		hashFunc = originalHashFunc
	}()

	// Override hashFunc to return a mock that returns an error
	hashFunc = func() hash.Hash {
		mock := &HashMock{}
		mock.WriteReturns(len(data), errors.New("mock write error"))
		mock.SumReturns(make([]byte, 32))
		mock.SizeReturns(32)
		mock.BlockSizeReturns(64)
		return mock
	}

	h := Hashable(data)
	assert.Panics(t, func() {
		h.Raw()
	}, "Should panic when hash.Write returns an error")
}

// TestHashable_Raw verifies Raw returns SHA256 hash of the data
func TestHashable_Raw(t *testing.T) {
	data := []byte("test data")
	h := Hashable(data)

	expected := sha256.Sum256(data)
	result := h.Raw()

	assert.Equal(t, expected[:], result)
}

// TestHashable_Raw_Empty verifies Raw returns nil for empty data
func TestHashable_Raw_Empty(t *testing.T) {
	h := Hashable([]byte{})
	result := h.Raw()

	assert.Nil(t, result)
}

// TestHashable_Raw_Nil verifies Raw returns nil for nil data
func TestHashable_Raw_Nil(t *testing.T) {
	var h Hashable
	result := h.Raw()

	assert.Nil(t, result)
}

// TestHashable_String verifies String returns base64 encoded hash
func TestHashable_String(t *testing.T) {
	data := []byte("test data")
	h := Hashable(data)

	hash := sha256.Sum256(data)
	expected := base64.StdEncoding.EncodeToString(hash[:])
	result := h.String()

	assert.Equal(t, expected, result)
}

// TestHashable_RawString verifies RawString returns string of raw hash bytes
func TestHashable_RawString(t *testing.T) {
	data := []byte("test data")
	h := Hashable(data)

	hash := sha256.Sum256(data)
	expected := string(hash[:])
	result := h.RawString()

	assert.Equal(t, expected, result)
}

// TestHashable_String_Empty verifies String handles empty data
func TestHashable_String_Empty(t *testing.T) {
	h := Hashable([]byte{})
	result := h.String()

	// Empty hashable returns base64 of nil which is empty string
	assert.Empty(t, result)
}

// TestHashable_Consistency verifies multiple calls return same result
func TestHashable_Consistency(t *testing.T) {
	data := []byte("consistent data")
	h := Hashable(data)

	raw1 := h.Raw()
	raw2 := h.Raw()
	str1 := h.String()
	str2 := h.String()

	assert.Equal(t, raw1, raw2)
	assert.Equal(t, str1, str2)
}

// TestHashable_RawString_Empty verifies RawString handles empty data
func TestHashable_RawString_Empty(t *testing.T) {
	h := Hashable([]byte{})
	result := h.RawString()

	// Empty hashable returns empty string since Raw() returns nil
	assert.Empty(t, result)
}

// TestHashable_RawString_Nil verifies RawString handles nil data
func TestHashable_RawString_Nil(t *testing.T) {
	var h Hashable
	result := h.RawString()

	// Nil hashable returns empty string since Raw() returns nil
	assert.Empty(t, result)
}

// TestHashable_String_Nil verifies String handles nil data
func TestHashable_String_Nil(t *testing.T) {
	var h Hashable
	result := h.String()

	// Nil hashable returns base64 of nil which is empty string
	assert.Empty(t, result)
}

// TestHashable_LargeData verifies handling of large data
func TestHashable_LargeData(t *testing.T) {
	// Create a large data set (1MB)
	data := make([]byte, 1024*1024)
	for i := range data {
		data[i] = byte(i % 256)
	}
	h := Hashable(data)

	// Should successfully hash large data
	result := h.Raw()
	assert.NotNil(t, result)
	assert.Len(t, result, 32) // SHA256 produces 32 bytes

	// Verify it's deterministic
	expected := sha256.Sum256(data)
	assert.Equal(t, expected[:], result)
}

// TestHashable_DifferentData verifies different data produces different hashes
func TestHashable_DifferentData(t *testing.T) {
	h1 := Hashable([]byte("data1"))
	h2 := Hashable([]byte("data2"))

	raw1 := h1.Raw()
	raw2 := h2.Raw()

	assert.NotEqual(t, raw1, raw2)
	assert.NotEqual(t, h1.String(), h2.String())
	assert.NotEqual(t, h1.RawString(), h2.RawString())
}

// TestHashable_SpecialCharacters verifies handling of special characters
func TestHashable_SpecialCharacters(t *testing.T) {
	data := []byte("test\x00\xFF\n\r\t data with special chars")
	h := Hashable(data)

	result := h.Raw()
	assert.NotNil(t, result)
	assert.Len(t, result, 32)

	// Verify String() produces valid base64
	str := h.String()
	decoded, err := base64.StdEncoding.DecodeString(str)
	require.NoError(t, err)
	assert.Equal(t, result, decoded)
}

// TestHashable_BinaryData verifies handling of binary data
func TestHashable_BinaryData(t *testing.T) {
	data := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}
	h := Hashable(data)

	result := h.Raw()
	assert.NotNil(t, result)
	assert.Len(t, result, 32)

	expected := sha256.Sum256(data)
	assert.Equal(t, expected[:], result)
}

// TestHashable_SingleByte verifies handling of single byte
func TestHashable_SingleByte(t *testing.T) {
	h := Hashable([]byte{0x42})

	result := h.Raw()
	assert.NotNil(t, result)
	assert.Len(t, result, 32)

	expected := sha256.Sum256([]byte{0x42})
	assert.Equal(t, expected[:], result)
}

// TestHashable_AllMethods verifies all methods work together correctly
func TestHashable_AllMethods(t *testing.T) {
	data := []byte("comprehensive test")
	h := Hashable(data)

	// Test Raw()
	raw := h.Raw()
	assert.NotNil(t, raw)
	assert.Len(t, raw, 32)

	// Test String()
	str := h.String()
	assert.NotEmpty(t, str)
	decoded, err := base64.StdEncoding.DecodeString(str)
	require.NoError(t, err)
	assert.Equal(t, raw, decoded)

	// Test RawString()
	rawStr := h.RawString()
	assert.NotEmpty(t, rawStr)
	assert.Equal(t, string(raw), rawStr)

	// Verify all methods are consistent
	assert.Equal(t, raw, h.Raw())
	assert.Equal(t, str, h.String())
	assert.Equal(t, rawStr, h.RawString())
}
