/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dbtest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTokenLockDBCases_Structure verifies that tokenLockDBCases is properly structured
func TestTokenLockDBCases_Structure(t *testing.T) {
	assert.NotEmpty(t, tokenLockDBCases, "tokenLockDBCases should not be empty")

	for _, tc := range tokenLockDBCases {
		assert.NotEmpty(t, tc.Name, "Test case name should not be empty")
		assert.NotNil(t, tc.Fn, "Test case function should not be nil")
	}
}

// TestTokenLockDBCases_ContainsTestFully verifies that TestFully is in the test cases
func TestTokenLockDBCases_ContainsTestFully(t *testing.T) {
	found := false
	for _, tc := range tokenLockDBCases {
		if tc.Name == "TestFully" {
			found = true
			assert.NotNil(t, tc.Fn, "TestFully function should not be nil")

			break
		}
	}
	require.True(t, found, "TestFully should be in tokenLockDBCases")
}

// TestTokenLockDBCases_AllNamesUnique verifies all test case names are unique
func TestTokenLockDBCases_AllNamesUnique(t *testing.T) {
	names := make(map[string]bool)
	for _, tc := range tokenLockDBCases {
		assert.False(t, names[tc.Name], "Duplicate test case name: %s", tc.Name)
		names[tc.Name] = true
	}
}

// TestTokenLockDBCases_Count verifies the expected number of test cases
func TestTokenLockDBCases_Count(t *testing.T) {
	assert.Len(t, tokenLockDBCases, 1, "Expected exactly 1 test case")
}
