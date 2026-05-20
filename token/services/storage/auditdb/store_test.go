/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package auditdb

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAcquireLocks_Success(t *testing.T) {
	store := &StoreService{
		eIDsLocks: sync.Map{},
	}
	ctx := context.Background()

	// Test acquiring locks for multiple enrollment IDs
	err := store.AcquireLocks(ctx, "anchor1", "alice", "bob", "charlie")
	require.NoError(t, err)

	// Verify that the anchor mapping was stored
	dedupBoxed, ok := store.eIDsLocks.Load("anchor1")
	require.True(t, ok, "anchor mapping should be stored")
	dedup := dedupBoxed.([]string)
	assert.ElementsMatch(t, []string{"alice", "bob", "charlie"}, dedup)

	// Clean up
	store.ReleaseLocks(ctx, "anchor1")
}

func TestAcquireLocks_Deduplication(t *testing.T) {
	store := &StoreService{
		eIDsLocks: sync.Map{},
	}
	ctx := context.Background()

	// Test with duplicate enrollment IDs
	err := store.AcquireLocks(ctx, "anchor2", "alice", "bob", "alice", "charlie", "bob")
	require.NoError(t, err)

	// Verify deduplication occurred
	dedupBoxed, ok := store.eIDsLocks.Load("anchor2")
	require.True(t, ok)
	dedup := dedupBoxed.([]string)
	assert.Len(t, dedup, 3, "duplicates should be removed")
	assert.ElementsMatch(t, []string{"alice", "bob", "charlie"}, dedup)

	// Clean up
	store.ReleaseLocks(ctx, "anchor2")
}

func TestAcquireLocks_Sorting(t *testing.T) {
	store := &StoreService{
		eIDsLocks: sync.Map{},
	}
	ctx := context.Background()

	// Test that enrollment IDs are sorted to prevent deadlocks
	err := store.AcquireLocks(ctx, "anchor3", "charlie", "alice", "bob")
	require.NoError(t, err)

	dedupBoxed, ok := store.eIDsLocks.Load("anchor3")
	require.True(t, ok)
	dedup := dedupBoxed.([]string)
	// Should be sorted alphabetically
	assert.Equal(t, []string{"alice", "bob", "charlie"}, dedup)

	// Clean up
	store.ReleaseLocks(ctx, "anchor3")
}

func TestAcquireLocks_ContextCancellation(t *testing.T) {
	store := &StoreService{
		eIDsLocks: sync.Map{},
	}

	// First, acquire a lock on "alice"
	ctx1 := context.Background()
	err := store.AcquireLocks(ctx1, "anchor_first", "alice")
	require.NoError(t, err)

	// Now try to acquire the same lock with a cancelled context
	ctx2, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err = store.AcquireLocks(ctx2, "anchor_second", "alice")
	require.Error(t, err, "should fail due to context cancellation")
	assert.Contains(t, err.Error(), "failed to acquire lock")

	// Verify that the second anchor was NOT stored
	_, ok := store.eIDsLocks.Load("anchor_second")
	assert.False(t, ok, "anchor should not be stored when lock acquisition fails")

	// Clean up
	store.ReleaseLocks(ctx1, "anchor_first")
}

func TestAcquireLocks_ContextTimeout(t *testing.T) {
	store := &StoreService{
		eIDsLocks: sync.Map{},
	}

	// First, acquire a lock on "bob"
	ctx1 := context.Background()
	err := store.AcquireLocks(ctx1, "anchor_holder", "bob")
	require.NoError(t, err)

	// Try to acquire the same lock with a short timeout
	ctx2, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err = store.AcquireLocks(ctx2, "anchor_timeout", "bob")
	require.Error(t, err, "should timeout waiting for lock")

	// Verify that the timeout anchor was NOT stored
	_, ok := store.eIDsLocks.Load("anchor_timeout")
	assert.False(t, ok, "anchor should not be stored when lock acquisition times out")

	// Clean up
	store.ReleaseLocks(ctx1, "anchor_holder")
}

func TestAcquireLocks_PartialAcquisitionRollback(t *testing.T) {
	store := &StoreService{
		eIDsLocks: sync.Map{},
	}

	// First, acquire a lock on "charlie"
	ctx1 := context.Background()
	err := store.AcquireLocks(ctx1, "anchor_blocker", "charlie")
	require.NoError(t, err)

	// Try to acquire locks on multiple IDs where one is already locked
	ctx2, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// This should fail because "charlie" is already locked
	// The sorted order will be: alice, bob, charlie
	err = store.AcquireLocks(ctx2, "anchor_partial", "alice", "bob", "charlie")
	require.Error(t, err, "should fail to acquire all locks")

	// Verify that locks on "alice" and "bob" were rolled back
	// We can test this by successfully acquiring them with a new anchor
	ctx3 := context.Background()
	err = store.AcquireLocks(ctx3, "anchor_verify", "alice", "bob")
	require.NoError(t, err, "alice and bob should be available (rollback successful)")

	// Clean up
	store.ReleaseLocks(ctx1, "anchor_blocker")
	store.ReleaseLocks(ctx3, "anchor_verify")
}

func TestAcquireLocks_ConcurrentNonOverlapping(t *testing.T) {
	store := &StoreService{
		eIDsLocks: sync.Map{},
	}
	ctx := context.Background()

	var wg sync.WaitGroup
	wg.Add(2)

	// Two goroutines acquiring non-overlapping locks should both succeed
	go func() {
		defer wg.Done()
		err := store.AcquireLocks(ctx, "anchor_concurrent1", "alice", "bob")
		assert.NoError(t, err)
		time.Sleep(100 * time.Millisecond)
		store.ReleaseLocks(ctx, "anchor_concurrent1")
	}()

	go func() {
		defer wg.Done()
		err := store.AcquireLocks(ctx, "anchor_concurrent2", "charlie", "dave")
		assert.NoError(t, err)
		time.Sleep(100 * time.Millisecond)
		store.ReleaseLocks(ctx, "anchor_concurrent2")
	}()

	wg.Wait()
}

func TestAcquireLocks_ConcurrentOverlapping(t *testing.T) {
	store := &StoreService{
		eIDsLocks: sync.Map{},
	}
	ctx := context.Background()

	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	// Multiple goroutines trying to acquire overlapping locks
	for i := range 5 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			anchor := fmt.Sprintf("anchor_overlap_%d", id)
			err := store.AcquireLocks(ctx, anchor, "shared_resource")
			if err == nil {
				mu.Lock()
				successCount++
				mu.Unlock()
				time.Sleep(50 * time.Millisecond)
				store.ReleaseLocks(ctx, anchor)
			}
		}(i)
	}

	wg.Wait()

	// All should eventually succeed (one at a time)
	assert.Equal(t, 5, successCount, "all goroutines should eventually acquire the lock")
}

func TestAcquireLocks_DeadlockPrevention(t *testing.T) {
	store := &StoreService{
		eIDsLocks: sync.Map{},
	}
	ctx := context.Background()

	var wg sync.WaitGroup
	wg.Add(2)

	// Two goroutines trying to acquire the same locks in different order
	// Sorting should prevent deadlock
	go func() {
		defer wg.Done()
		err := store.AcquireLocks(ctx, "anchor_deadlock1", "alice", "bob")
		assert.NoError(t, err)
		time.Sleep(100 * time.Millisecond)
		store.ReleaseLocks(ctx, "anchor_deadlock1")
	}()

	go func() {
		defer wg.Done()
		// Different order, but sorting will make it the same
		err := store.AcquireLocks(ctx, "anchor_deadlock2", "bob", "alice")
		assert.NoError(t, err)
		time.Sleep(100 * time.Millisecond)
		store.ReleaseLocks(ctx, "anchor_deadlock2")
	}()

	// Use a timeout to detect if deadlock occurs
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success - no deadlock
	case <-time.After(5 * time.Second):
		t.Fatal("Deadlock detected - goroutines did not complete")
	}
}

func TestAcquireLocks_EmptyEnrollmentIDs(t *testing.T) {
	store := &StoreService{
		eIDsLocks: sync.Map{},
	}
	ctx := context.Background()

	// Test with no enrollment IDs
	err := store.AcquireLocks(ctx, "anchor_empty")
	require.NoError(t, err)

	// Verify that the anchor mapping was stored (with empty slice)
	dedupBoxed, ok := store.eIDsLocks.Load("anchor_empty")
	require.True(t, ok)
	dedup := dedupBoxed.([]string)
	assert.Empty(t, dedup)

	// Clean up
	store.ReleaseLocks(ctx, "anchor_empty")
}

func TestAcquireLocks_SingleEnrollmentID(t *testing.T) {
	store := &StoreService{
		eIDsLocks: sync.Map{},
	}
	ctx := context.Background()

	// Test with a single enrollment ID
	err := store.AcquireLocks(ctx, "anchor_single", "alice")
	require.NoError(t, err)

	dedupBoxed, ok := store.eIDsLocks.Load("anchor_single")
	require.True(t, ok)
	dedup := dedupBoxed.([]string)
	assert.Equal(t, []string{"alice"}, dedup)

	// Clean up
	store.ReleaseLocks(ctx, "anchor_single")
}

func TestReleaseLocks_NonExistentAnchor(t *testing.T) {
	store := &StoreService{
		eIDsLocks: sync.Map{},
	}
	ctx := context.Background()

	// Releasing locks for a non-existent anchor should not panic
	store.ReleaseLocks(ctx, "non_existent_anchor")
	// If we reach here without panic, test passes
}

func TestAcquireAndReleaseLocks_Integration(t *testing.T) {
	store := &StoreService{
		eIDsLocks: sync.Map{},
	}
	ctx := context.Background()

	// Acquire locks
	err := store.AcquireLocks(ctx, "anchor_integration", "alice", "bob", "charlie")
	require.NoError(t, err)

	// Verify locks are held
	_, ok := store.eIDsLocks.Load("anchor_integration")
	require.True(t, ok)

	// Release locks
	store.ReleaseLocks(ctx, "anchor_integration")

	// Verify anchor mapping is removed
	_, ok = store.eIDsLocks.Load("anchor_integration")
	assert.False(t, ok, "anchor mapping should be removed after release")

	// Verify we can acquire the same locks again
	err = store.AcquireLocks(ctx, "anchor_integration2", "alice", "bob", "charlie")
	require.NoError(t, err, "should be able to re-acquire released locks")

	// Clean up
	store.ReleaseLocks(ctx, "anchor_integration2")
}

// Made with Bob
