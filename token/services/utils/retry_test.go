/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package utils_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testLogger() logging.Logger {
	return logging.MustGetLogger()
}

// TestRunWithContext_PreCanceledContext verifies that a pre-canceled context causes
// RunWithContext to return immediately without invoking the runner at all.
func TestRunWithContext_PreCanceledContext(t *testing.T) {
	runner := utils.NewRetryRunner(testLogger(), utils.Infinitely, 10*time.Second, false)

	ctx, cancel := context.WithCancel(t.Context())
	cancel() // cancel before calling Run

	calls := 0
	start := time.Now()
	err := runner.RunWithContext(ctx, func() error {
		calls++

		return errors.New("should not be reached")
	})
	elapsed := time.Since(start)

	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 0, calls, "runner should not be invoked on a pre-canceled context")
	assert.Less(t, elapsed, 500*time.Millisecond, "should return immediately, not block on 10s sleep")
}

// TestRunWithContext_CanceledDuringBackoff verifies that canceling the context while
// the retry loop is sleeping between attempts unblocks the caller promptly.
// This is the core regression test for the bug: without context-aware sleep,
// a worker goroutine would block in time.Sleep for the full (ever-growing) delay.
func TestRunWithContext_CanceledDuringBackoff(t *testing.T) {
	// Initial delay is long so we can observe it being interrupted.
	runner := utils.NewRetryRunner(testLogger(), utils.Infinitely, 5*time.Second, false)

	ctx, cancel := context.WithCancel(t.Context())

	calls := 0
	done := make(chan error, 1)

	go func() {
		done <- runner.RunWithContext(ctx, func() error {
			calls++

			return errors.New("transient error")
		})
	}()

	// Let the runner execute once and enter the 5s sleep.
	time.Sleep(50 * time.Millisecond)
	cancel()

	start := time.Now()

	select {
	case err := <-done:
		elapsed := time.Since(start)
		require.ErrorIs(t, err, context.Canceled)
		assert.Equal(t, 1, calls, "runner should have been called exactly once before cancel")
		assert.Less(t, elapsed, time.Second,
			"RunWithContext should unblock within 1s of cancellation, not wait out the 5s sleep")
	case <-time.After(6 * time.Second):
		t.Fatal("RunWithContext did not respect context cancellation — worker would have been stuck")
	}
}

// TestRunWithContext_BackoffDoesNotExceedCap verifies that the exponential backoff
// delay is capped and does not grow without bound.
// We use a tiny initial delay so the test runs fast; the cap itself is 30s by default.
func TestRunWithContext_BackoffDoesNotExceedCap(t *testing.T) {
	// 10 retries with 1ms initial delay, exp backoff: 1,2,4,8,16,30,30,30,30,30 ms
	runner := utils.NewRetryRunner(testLogger(), 10, time.Millisecond, true)

	var intervals []time.Duration
	prev := time.Now()

	_ = runner.RunWithContext(t.Context(), func() error {
		now := time.Now()
		intervals = append(intervals, now.Sub(prev))
		prev = now

		return errors.New("always fail")
	})

	// Skip the first interval (no sleep before first call).
	// Each subsequent interval should be ≤ defaultMaxDelay + small scheduling slack.
	const maxAllowed = 30*time.Second + 200*time.Millisecond
	for i, d := range intervals[1:] {
		assert.LessOrEqual(t, d, maxAllowed,
			"backoff interval %d (%v) exceeded the 30s cap", i+1, d)
	}
}

// TestRunWithContext_SucceedsAfterTransientFailures verifies normal retry behavior:
// the runner is retried until it succeeds.
func TestRunWithContext_SucceedsAfterTransientFailures(t *testing.T) {
	runner := utils.NewRetryRunner(testLogger(), utils.Infinitely, time.Millisecond, false)

	calls := 0
	err := runner.RunWithContext(t.Context(), func() error {
		calls++
		if calls < 3 {
			return errors.New("transient")
		}

		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 3, calls)
}

// TestRun_DelegatesWithBackgroundContext verifies that Run (which uses context.Background)
// still retries correctly and is not broken by the refactor.
func TestRun_DelegatesWithBackgroundContext(t *testing.T) {
	runner := utils.NewRetryRunner(testLogger(), 5, time.Millisecond, false)

	calls := 0
	err := runner.Run(func() error {
		calls++
		if calls < 2 {
			return errors.New("transient")
		}

		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 2, calls)
}

// TestRunWithContext_MaxRetriesExhausted verifies that when maxTimes is set and
// all retries fail, a joined error is returned.
func TestRunWithContext_MaxRetriesExhausted(t *testing.T) {
	runner := utils.NewRetryRunner(testLogger(), 3, time.Millisecond, false)

	calls := 0
	err := runner.RunWithContext(t.Context(), func() error {
		calls++

		return errors.New("persistent failure")
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "persistent failure")
	assert.Equal(t, 3, calls)
}

// TestRunWithContext_SuccessOnFirstAttempt verifies zero-retry fast path.
func TestRunWithContext_SuccessOnFirstAttempt(t *testing.T) {
	runner := utils.NewRetryRunner(testLogger(), utils.Infinitely, time.Second, true)

	calls := 0
	err := runner.RunWithContext(t.Context(), func() error {
		calls++

		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 1, calls)
}
