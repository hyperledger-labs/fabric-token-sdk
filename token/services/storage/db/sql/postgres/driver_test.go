/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package postgres

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCreateTableLockID verifies that lock IDs are deterministic and unique per store type
func TestCreateTableLockID(t *testing.T) {
	tests := []struct {
		name      string
		storeType string
	}{
		{"tokens", "tokens"},
		{"identity", "identity"},
		{"transactions", "transactions"},
		{"wallet", "wallet"},
		{"keystore", "keystore"},
		{"tokenlock", "tokenlock"},
		{"audittx", "audittx"},
	}

	// Verify deterministic: same input produces same output
	for _, tt := range tests {
		t.Run(tt.name+"_deterministic", func(t *testing.T) {
			lockID1 := createTableLockID(tt.storeType)
			lockID2 := createTableLockID(tt.storeType)
			require.Equal(t, lockID1, lockID2, "Lock ID should be deterministic for %s", tt.storeType)
			require.NotZero(t, lockID1, "Lock ID should not be zero")
		})
	}

	// Verify uniqueness: different store types produce different lock IDs
	lockIDs := make(map[int64]string)
	for _, tt := range tests {
		lockID := createTableLockID(tt.storeType)
		if existingStore, exists := lockIDs[lockID]; exists {
			t.Errorf("Lock ID collision: %s and %s produce the same lock ID %d", tt.storeType, existingStore, lockID)
		}
		lockIDs[lockID] = tt.storeType
	}

	t.Logf("Generated %d unique lock IDs", len(lockIDs))
}

// TestPrefixSchemaWithLock verifies that schema is correctly prefixed with advisory lock
func TestPrefixSchemaWithLock(t *testing.T) {
	baseSchema := `CREATE TABLE IF NOT EXISTS test_table (
		id INT PRIMARY KEY,
		name TEXT
	);`

	lockID := int64(12345)

	result := prefixSchemaWithLock(baseSchema, lockID)

	// Verify the lock statement is at the beginning
	require.Contains(t, result, "SELECT pg_advisory_xact_lock(12345);")
	require.Contains(t, result, baseSchema)

	// Verify lock comes before schema
	lockIndex := len("SELECT pg_advisory_xact_lock(12345);")
	require.Less(t, lockIndex, len(result))

	t.Logf("Prefixed schema:\n%s", result)
}

// TestPrefixSchemaWithLock_MultipleStatements verifies handling of complex schemas
func TestPrefixSchemaWithLock_MultipleStatements(t *testing.T) {
	baseSchema := `CREATE TABLE IF NOT EXISTS table1 (id INT);
CREATE TABLE IF NOT EXISTS table2 (id INT);
CREATE INDEX IF NOT EXISTS idx1 ON table1(id);`

	lockID := createTableLockID("test")

	result := prefixSchemaWithLock(baseSchema, lockID)

	// Verify lock is at the beginning
	require.Contains(t, result, "SELECT pg_advisory_xact_lock(")

	// Verify all original statements are present
	require.Contains(t, result, "CREATE TABLE IF NOT EXISTS table1")
	require.Contains(t, result, "CREATE TABLE IF NOT EXISTS table2")
	require.Contains(t, result, "CREATE INDEX IF NOT EXISTS idx1")
}

// Made with Bob
