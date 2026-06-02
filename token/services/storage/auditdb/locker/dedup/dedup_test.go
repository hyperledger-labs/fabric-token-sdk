/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dedup

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAndSort(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: []string{},
		},
		{
			name:     "empty input",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "single element",
			input:    []string{"alice"},
			expected: []string{"alice"},
		},
		{
			name:     "already sorted and unique",
			input:    []string{"alice", "bob", "charlie"},
			expected: []string{"alice", "bob", "charlie"},
		},
		{
			name:     "unsorted is sorted ascending",
			input:    []string{"charlie", "alice", "bob"},
			expected: []string{"alice", "bob", "charlie"},
		},
		{
			name:     "duplicates removed",
			input:    []string{"bob", "alice", "bob", "alice", "bob"},
			expected: []string{"alice", "bob"},
		},
		{
			name:     "intersecting sets resolve to the same order (deadlock-free invariant)",
			input:    []string{"e2", "e1"},
			expected: []string{"e1", "e2"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := AndSort(tc.input)
			assert.Equal(t, tc.expected, got)
		})
	}
}

// TestAndSort_CanonicalOrderIsStable checks the property the lockers rely on:
// regardless of the input order, two slices with the same set of enrollment IDs
// produce the identical acquisition order, so they cannot deadlock.
func TestAndSort_CanonicalOrderIsStable(t *testing.T) {
	a := AndSort([]string{"x", "y", "z"})
	b := AndSort([]string{"z", "y", "x"})
	c := AndSort([]string{"y", "z", "x", "y"})

	assert.Equal(t, a, b)
	assert.Equal(t, a, c)
}
