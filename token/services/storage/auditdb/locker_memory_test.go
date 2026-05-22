/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package auditdb

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryLocker_AcquireAndRelease(t *testing.T) {
	l := newMemoryLocker()
	ctx := context.Background()

	require.NoError(t, l.AcquireLocks(ctx, "anchor1", "alice", "bob"))
	l.ReleaseLocks(ctx, "anchor1")
}

func TestMemoryLocker_ReleaseUnknownAnchor(t *testing.T) {
	l := newMemoryLocker()
	l.ReleaseLocks(context.Background(), "nonexistent")
}

func TestMemoryLocker_AssertLocksHeld_AlwaysSucceeds(t *testing.T) {
	l := newMemoryLocker()
	ctx := context.Background()
	require.NoError(t, l.AssertLocksHeld(ctx, "no-such-anchor"))
}

func TestMemoryLocker_NonOverlappingConcurrency(t *testing.T) {
	l := newMemoryLocker()
	ctx := context.Background()
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		assert.NoError(t, l.AcquireLocks(ctx, "a1", "alice"))
		time.Sleep(10 * time.Millisecond)
		l.ReleaseLocks(ctx, "a1")
	}()
	go func() {
		defer wg.Done()
		assert.NoError(t, l.AcquireLocks(ctx, "a2", "bob"))
		time.Sleep(10 * time.Millisecond)
		l.ReleaseLocks(ctx, "a2")
	}()

	wg.Wait()
}

func TestMemoryLocker_OverlappingBlocks(t *testing.T) {
	l := newMemoryLocker()
	ctx := context.Background()
	acquired := make(chan struct{})

	require.NoError(t, l.AcquireLocks(ctx, "a1", "alice"))

	go func() {
		_ = l.AcquireLocks(ctx, "a2", "alice")
		close(acquired)
	}()

	select {
	case <-acquired:
		t.Fatal("second acquire should block while first holds the lock")
	case <-time.After(50 * time.Millisecond):
	}

	l.ReleaseLocks(ctx, "a1")

	select {
	case <-acquired:
	case <-time.After(time.Second):
		t.Fatal("second acquire should proceed after first releases")
	}
	l.ReleaseLocks(ctx, "a2")
}

func TestMemoryLocker_DeduplicateAndSort(t *testing.T) {
	l := newMemoryLocker()
	ctx := context.Background()
	require.NoError(t, l.AcquireLocks(ctx, "a1", "bob", "alice", "alice"))
	l.ReleaseLocks(ctx, "a1")
}

func TestMemoryLocker_EmptyEIDs(t *testing.T) {
	l := newMemoryLocker()
	ctx := context.Background()
	require.NoError(t, l.AcquireLocks(ctx, "a1"))
	l.ReleaseLocks(ctx, "a1")
}

func TestMemoryLocker_DeadlockPrevention(t *testing.T) {
	l := newMemoryLocker()
	ctx := context.Background()
	done := make(chan struct{})

	go func() {
		defer close(done)
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			_ = l.AcquireLocks(ctx, "a1", "alice", "bob")
			time.Sleep(5 * time.Millisecond)
			l.ReleaseLocks(ctx, "a1")
		}()
		go func() {
			defer wg.Done()
			_ = l.AcquireLocks(ctx, "a2", "bob", "alice")
			time.Sleep(5 * time.Millisecond)
			l.ReleaseLocks(ctx, "a2")
		}()
		wg.Wait()
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("deadlock detected")
	}
}

func TestStoreService_WithLocker(t *testing.T) {
	// Verify that the WithLocker option is respected.
	custom := &stubLocker{}
	s, err := NewStoreService(nil, WithLocker(custom))
	require.NoError(t, err)
	assert.Same(t, custom, s.locker)
}

type stubLocker struct{}

func (s *stubLocker) AcquireLocks(context.Context, string, ...string) error { return nil }
func (s *stubLocker) ReleaseLocks(context.Context, string)                  {}
func (s *stubLocker) AssertLocksHeld(context.Context, string) error         { return nil }
