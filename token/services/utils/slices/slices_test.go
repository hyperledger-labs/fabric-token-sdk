/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

// This file tests slices.go which provides generic slice utilities.
// Tests cover element retrieval and pointer slice generation.
package slices

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGetUnique verifies GetUnique returns the first element
func TestGetUnique(t *testing.T) {
	t.Run("string slice", func(t *testing.T) {
		slice := []string{"first", "second", "third"}
		result := GetUnique(slice)
		assert.Equal(t, "first", result)
	})

	t.Run("int slice", func(t *testing.T) {
		slice := []int{1, 2, 3}
		result := GetUnique(slice)
		assert.Equal(t, 1, result)
	})

	t.Run("single element", func(t *testing.T) {
		slice := []string{"only"}
		result := GetUnique(slice)
		assert.Equal(t, "only", result)
	})
}

// TestGetAny verifies GetAny returns the first element
func TestGetAny(t *testing.T) {
	t.Run("string slice", func(t *testing.T) {
		slice := []string{"first", "second", "third"}
		result := GetAny(slice)
		assert.Equal(t, "first", result)
	})

	t.Run("int slice", func(t *testing.T) {
		slice := []int{10, 20, 30}
		result := GetAny(slice)
		assert.Equal(t, 10, result)
	})
}

// TestGetFirst verifies GetFirst returns the first element
func TestGetFirst(t *testing.T) {
	t.Run("string slice", func(t *testing.T) {
		slice := []string{"alpha", "beta", "gamma"}
		result := GetFirst(slice)
		assert.Equal(t, "alpha", result)
	})

	t.Run("struct slice", func(t *testing.T) {
		type TestStruct struct {
			Value int
		}
		slice := []TestStruct{{Value: 1}, {Value: 2}}
		result := GetFirst(slice)
		assert.Equal(t, TestStruct{Value: 1}, result)
	})
}

// TestGenericSliceOfPointers verifies GenericSliceOfPointers creates a slice of pointers
func TestGenericSliceOfPointers(t *testing.T) {
	t.Run("int pointers", func(t *testing.T) {
		size := 5
		slice := GenericSliceOfPointers[int](size)

		assert.Len(t, slice, size)
		for i, ptr := range slice {
			assert.NotNil(t, ptr, "pointer at index %d should not be nil", i)
			assert.Equal(t, 0, *ptr, "pointer should point to zero value")
		}
	})

	t.Run("string pointers", func(t *testing.T) {
		size := 3
		slice := GenericSliceOfPointers[string](size)

		assert.Len(t, slice, size)
		for i, ptr := range slice {
			assert.NotNil(t, ptr, "pointer at index %d should not be nil", i)
			assert.Empty(t, *ptr, "pointer should point to zero value")
		}
	})

	t.Run("struct pointers", func(t *testing.T) {
		type TestStruct struct {
			Field1 string
			Field2 int
		}
		size := 2
		slice := GenericSliceOfPointers[TestStruct](size)

		assert.Len(t, slice, size)
		for i, ptr := range slice {
			assert.NotNil(t, ptr, "pointer at index %d should not be nil", i)
			assert.Equal(t, TestStruct{}, *ptr, "pointer should point to zero value")
		}
	})

	t.Run("zero size", func(t *testing.T) {
		slice := GenericSliceOfPointers[int](0)
		assert.Empty(t, slice)
	})

	t.Run("pointers are independent", func(t *testing.T) {
		slice := GenericSliceOfPointers[int](3)

		// Modify one pointer
		*slice[0] = 10
		*slice[1] = 20
		*slice[2] = 30

		// Verify they're independent
		assert.Equal(t, 10, *slice[0])
		assert.Equal(t, 20, *slice[1])
		assert.Equal(t, 30, *slice[2])
	})
}

// Made with Bob
