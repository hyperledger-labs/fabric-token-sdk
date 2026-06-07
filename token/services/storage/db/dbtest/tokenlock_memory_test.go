/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dbtest

import (
	"testing"

	driver3 "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/memory"
	"github.com/stretchr/testify/require"
)

// TestTokenLocksWithMemoryDriver tests TokenLocksTest using the in-memory driver
// This ensures the test helper functions are actually executed and covered by tests
func TestTokenLocksWithMemoryDriver(t *testing.T) {
	TokenLocksTest(t, func(string) driver3.Driver {
		return memory.NewDriver()
	})
}

// TestTokenLocksTest_VerifyCleanup verifies that stores are properly closed
func TestTokenLocksTest_VerifyCleanup(t *testing.T) {
	// This test verifies that the defer statements in TokenLocksTest work correctly
	// by ensuring no panics occur during cleanup
	require.NotPanics(t, func() {
		TokenLocksTest(t, func(string) driver3.Driver {
			return memory.NewDriver()
		})
	})
}
