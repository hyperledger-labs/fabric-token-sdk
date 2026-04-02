/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package finality_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx/dep/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx/finality"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// stubTxDB is a minimal transactionDB implementation for testing.
type stubTxDB struct {
	getTokenRequestFn func(ctx context.Context, txID string) ([]byte, error)
	setStatusFn       func(ctx context.Context, txID string, status storage.TxStatus, message string) error
}

func (s *stubTxDB) GetTokenRequest(ctx context.Context, txID string) ([]byte, error) {
	if s.getTokenRequestFn != nil {
		return s.getTokenRequestFn(ctx, txID)
	}

	return nil, nil
}

func (s *stubTxDB) SetStatus(ctx context.Context, txID string, status storage.TxStatus, message string) error {
	if s.setStatusFn != nil {
		return s.setStatusFn(ctx, txID, status, message)
	}

	return nil
}

func noopTracer() trace.Tracer {
	return noop.NewTracerProvider().Tracer("")
}

func listenerLogger() logging.Logger {
	return logging.MustGetLogger()
}

// newTestListener builds a Listener wired with the given ttxDB stub.
// tokens.Service is nil because the test cases below never reach the token-append path.
func newTestListener(t *testing.T, db *stubTxDB) *finality.Listener {
	t.Helper()

	return finality.NewListener(
		listenerLogger(),
		&mock.Network{},
		"test-namespace",
		&mock.TokenManagementServiceProvider{},
		token.TMSID{Network: "n", Channel: "c", Namespace: "ns"},
		db,
		nil, // tokens.Service — not accessed for network.Invalid status
		noopTracer(),
		nil, // metricsProvider — noop fallback
	)
}

// TestOnStatus_ContextCanceledDuringRetry is the primary regression test.
//
// Setup: ttxDB.SetStatus always returns a transient error, so the inner retryRunner
// would spin forever under the old code (unbounded exponential backoff, no context check).
//
// The fix: RunWithContext is used, so canceling the context unblocks the sleeping
// retry loop and OnStatus returns promptly.
func TestOnStatus_ContextCanceledDuringRetry(t *testing.T) {
	var setCalls atomic.Int32
	db := &stubTxDB{
		setStatusFn: func(_ context.Context, _ string, _ storage.TxStatus, _ string) error {
			setCalls.Add(1)

			return errors.New("db unavailable")
		},
	}
	l := newTestListener(t, db)

	ctx, cancel := context.WithCancel(t.Context())

	done := make(chan struct{})

	go func() {
		defer close(done)
		// network.Invalid → runOnStatus sets txStatus=Deleted, then calls SetStatus.
		// SetStatus always fails → retry loop engages.
		l.OnStatus(ctx, "tx1", network.Invalid, "validation failed", nil)
	}()

	// Allow at least one SetStatus call before canceling.
	require.Eventually(t, func() bool { return setCalls.Load() >= 1 }, time.Second, 10*time.Millisecond)
	cancel()

	select {
	case <-done:
		// good: OnStatus returned after context was canceled
	case <-time.After(5 * time.Second):
		t.Fatal("OnStatus did not return after context cancellation — worker goroutine would be permanently stuck")
	}
}

// TestOnStatus_SucceedsAfterTransientError verifies that OnStatus retries correctly
// and completes successfully once the transient error resolves.
func TestOnStatus_SucceedsAfterTransientError(t *testing.T) {
	var setCalls atomic.Int32
	db := &stubTxDB{
		setStatusFn: func(_ context.Context, _ string, _ storage.TxStatus, _ string) error {
			n := setCalls.Add(1)
			if n < 3 {
				return errors.New("transient db error")
			}

			return nil
		},
	}
	l := newTestListener(t, db)

	done := make(chan struct{})

	go func() {
		defer close(done)
		l.OnStatus(t.Context(), "tx1", network.Invalid, "validation failed", nil)
	}()

	select {
	case <-done:
		assert.GreaterOrEqual(t, int(setCalls.Load()), 3, "should have retried at least 3 times")
	case <-time.After(5 * time.Second):
		t.Fatal("OnStatus did not complete after transient errors resolved")
	}
}

// TestOnStatus_CompletesImmediatelyOnSuccess verifies the happy path:
// when SetStatus succeeds on the first try, OnStatus returns without any retries.
func TestOnStatus_CompletesImmediatelyOnSuccess(t *testing.T) {
	var setCalls atomic.Int32
	db := &stubTxDB{
		setStatusFn: func(_ context.Context, _ string, _ storage.TxStatus, _ string) error {
			setCalls.Add(1)

			return nil
		},
	}
	l := newTestListener(t, db)

	l.OnStatus(t.Context(), "tx1", network.Invalid, "invalid", nil)

	assert.Equal(t, int32(1), setCalls.Load())
}

// TestOnStatus_ConcurrentCallsAreIndependent verifies that concurrent OnStatus calls
// for different transactions do not interfere with each other.
func TestOnStatus_ConcurrentCallsAreIndependent(t *testing.T) {
	var mu sync.Mutex

	statusSet := map[string]bool{}

	db := &stubTxDB{
		setStatusFn: func(_ context.Context, txID string, _ storage.TxStatus, _ string) error {
			mu.Lock()
			statusSet[txID] = true
			mu.Unlock()

			return nil
		},
	}
	l := newTestListener(t, db)

	txIDs := []string{"tx1", "tx2", "tx3", "tx4", "tx5"}

	var wg sync.WaitGroup

	for _, txID := range txIDs {
		wg.Add(1)

		go func(id string) {
			defer wg.Done()
			l.OnStatus(t.Context(), id, network.Invalid, "invalid", nil)
		}(txID)
	}

	wg.Wait()

	mu.Lock()
	defer mu.Unlock()

	for _, id := range txIDs {
		assert.True(t, statusSet[id], "tx %s should have had its status set", id)
	}
}

// TestOnStatus_PreCanceledContextReturnsImmediately verifies that a pre-canceled context
// causes OnStatus to return before the first retry sleep, not after it.
func TestOnStatus_PreCanceledContextReturnsImmediately(t *testing.T) {
	db := &stubTxDB{
		setStatusFn: func(_ context.Context, _ string, _ storage.TxStatus, _ string) error {
			return errors.New("always fails")
		},
	}
	l := newTestListener(t, db)

	ctx, cancel := context.WithCancel(t.Context())
	cancel() // cancel before calling OnStatus

	start := time.Now()
	l.OnStatus(ctx, "tx1", network.Invalid, "invalid", nil)
	elapsed := time.Since(start)

	// Should not block for the 1s initial retry delay.
	assert.Less(t, elapsed, 500*time.Millisecond,
		"OnStatus should return promptly when context is already canceled")
}

// TestOnStatus_StatusSetToDeletedForInvalidTx verifies that for an Invalid network
// status, the local DB is updated to Deleted.
func TestOnStatus_StatusSetToDeletedForInvalidTx(t *testing.T) {
	var capturedStatus storage.TxStatus
	db := &stubTxDB{
		setStatusFn: func(_ context.Context, _ string, s storage.TxStatus, _ string) error {
			capturedStatus = s

			return nil
		},
	}
	l := newTestListener(t, db)

	l.OnStatus(t.Context(), "tx1", network.Invalid, "rejected", nil)

	require.Equal(t, storage.Deleted, capturedStatus,
		"an Invalid network status should map to Deleted in local storage")
}
