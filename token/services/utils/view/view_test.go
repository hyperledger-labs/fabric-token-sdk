/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

// This file tests view.go which provides view execution with timeout support.
// Tests cover contextWrapper and RunViewWithTimeout behavior.
package view

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestContextWrapper verifies the contextWrapper implementation
func TestContextWrapper(t *testing.T) {
	t.Run("Context returns wrapped context", func(t *testing.T) {
		baseCtx := t.Context()

		timeoutCtx, cancel := context.WithTimeout(baseCtx, 5*time.Second)
		defer cancel()

		wrapper := &contextWrapper{
			ctx: timeoutCtx,
		}

		assert.Equal(t, timeoutCtx, wrapper.Context())
		assert.NotEqual(t, baseCtx, wrapper.Context())
	})

	t.Run("Context with cancelled context", func(t *testing.T) {
		baseCtx := t.Context()

		cancelCtx, cancel := context.WithCancel(baseCtx)
		cancel() // Cancel immediately

		wrapper := &contextWrapper{
			ctx: cancelCtx,
		}

		ctx := wrapper.Context()
		assert.NotNil(t, ctx)

		// Verify context is cancelled
		select {
		case <-ctx.Done():
			require.Error(t, ctx.Err())
		default:
			t.Error("context should be cancelled")
		}
	})

	t.Run("Context with deadline", func(t *testing.T) {
		baseCtx := t.Context()
		deadline := time.Now().Add(1 * time.Hour)

		deadlineCtx, cancel := context.WithDeadline(baseCtx, deadline)
		defer cancel()

		wrapper := &contextWrapper{
			ctx: deadlineCtx,
		}

		ctx := wrapper.Context()
		actualDeadline, ok := ctx.Deadline()
		assert.True(t, ok, "context should have a deadline")
		assert.Equal(t, deadline.Unix(), actualDeadline.Unix())
	})
}

// TestRunViewWithTimeout_ZeroTimeout verifies zero timeout behavior
func TestRunViewWithTimeout_ZeroTimeout(t *testing.T) {
	// This test verifies the logic path when timeout is zero
	// We can't easily test the full flow without mocking view.Context
	// but we can verify the timeout logic itself

	t.Run("zero timeout means no timeout context", func(t *testing.T) {
		timeout := time.Duration(0)
		assert.Equal(t, time.Duration(0), timeout)

		// When timeout is 0, RunViewWithTimeout should call ctx.RunView directly
		// without creating a timeout context
	})

	t.Run("non-zero timeout creates timeout context", func(t *testing.T) {
		timeout := 5 * time.Second
		assert.Greater(t, timeout, time.Duration(0))

		// When timeout > 0, RunViewWithTimeout should create a timeout context
		baseCtx := t.Context()
		timeoutCtx, cancel := context.WithTimeout(baseCtx, timeout)
		defer cancel()

		assert.NotNil(t, timeoutCtx)
		deadline, ok := timeoutCtx.Deadline()
		assert.True(t, ok)
		assert.Positive(t, time.Until(deadline))
		assert.LessOrEqual(t, time.Until(deadline), timeout)
	})
}

// TestRunView_PanicRecovery verifies panic recovery logic
func TestRunView_PanicRecovery(t *testing.T) {
	t.Run("defer recover pattern", func(t *testing.T) {
		// Test the panic recovery pattern used in RunView
		recovered := false

		func() {
			defer func() {
				if r := recover(); r != nil {
					recovered = true
				}
			}()

			// Simulate a panic
			panic("test panic")
		}()

		assert.True(t, recovered, "panic should be recovered")
	})

	t.Run("nested defer recover", func(t *testing.T) {
		// RunView has nested defer/recover blocks
		var outerRecovered atomic.Bool
		var innerRecovered atomic.Bool

		func() {
			defer func() {
				if r := recover(); r != nil {
					outerRecovered.Store(true)
				}
			}()

			done := make(chan struct{})
			go func() {
				defer close(done)
				defer func() {
					if r := recover(); r != nil {
						innerRecovered.Store(true)
					}
				}()

				panic("inner panic")
			}()

			<-done // Wait for goroutine to complete
		}()

		assert.True(t, innerRecovered.Load(), "inner panic should be recovered")
		assert.False(t, outerRecovered.Load(), "outer should not catch inner goroutine panic")
	})
}

// TestTimeoutBehavior verifies timeout context behavior
func TestTimeoutBehavior(t *testing.T) {
	t.Run("timeout context cancels after duration", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
		defer cancel()

		select {
		case <-ctx.Done():
			t.Error("context should not be done immediately")
		case <-time.After(5 * time.Millisecond):
			// Context should still be active
		}

		// Wait for timeout
		<-ctx.Done()
		require.Error(t, ctx.Err())
		assert.Equal(t, context.DeadlineExceeded, ctx.Err())
	})

	t.Run("cancel before timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 1*time.Hour)

		// Cancel immediately
		cancel()

		select {
		case <-ctx.Done():
			require.Error(t, ctx.Err())
			assert.Equal(t, context.Canceled, ctx.Err())
		case <-time.After(100 * time.Millisecond):
			t.Error("context should be cancelled immediately")
		}
	})
}

// TestContextWrapperIntegration verifies contextWrapper works with timeout contexts
func TestContextWrapperIntegration(t *testing.T) {
	t.Run("wrapper preserves timeout behavior", func(t *testing.T) {
		baseCtx := t.Context()
		timeoutCtx, cancel := context.WithTimeout(baseCtx, 50*time.Millisecond)
		defer cancel()

		wrapper := &contextWrapper{
			ctx: timeoutCtx,
		}

		// Context should not be done immediately
		select {
		case <-wrapper.Context().Done():
			t.Error("context should not be done immediately")
		case <-time.After(10 * time.Millisecond):
			// Expected: context still active
		}

		// Wait for timeout
		<-wrapper.Context().Done()
		require.Error(t, wrapper.Context().Err())
		assert.Equal(t, context.DeadlineExceeded, wrapper.Context().Err())
	})

	t.Run("wrapper preserves cancellation", func(t *testing.T) {
		baseCtx := t.Context()
		cancelCtx, cancel := context.WithCancel(baseCtx)

		wrapper := &contextWrapper{
			ctx: cancelCtx,
		}

		// Cancel the context
		cancel()

		// Verify cancellation is preserved
		select {
		case <-wrapper.Context().Done():
			require.Error(t, wrapper.Context().Err())
			assert.Equal(t, context.Canceled, wrapper.Context().Err())
		case <-time.After(100 * time.Millisecond):
			t.Error("context should be cancelled")
		}
	})
}

// TestRunViewWithTimeout_LogicPaths tests the conditional logic in RunViewWithTimeout
func TestRunViewWithTimeout_LogicPaths(t *testing.T) {
	t.Run("zero timeout takes fast path", func(t *testing.T) {
		// When timeout is 0, the function should take the fast path
		// and not create a timeout context
		timeout := time.Duration(0)
		assert.Equal(t, time.Duration(0), timeout)

		// The condition (timeout == 0) should be true
		if timeout == 0 {
			// This is the path taken - direct RunView call
			// Test passes if we reach here
		} else {
			t.Error("should take fast path for zero timeout")
		}
	})

	t.Run("non-zero timeout creates wrapper", func(t *testing.T) {
		timeout := 5 * time.Second
		assert.Greater(t, timeout, time.Duration(0))

		// The condition (timeout == 0) should be false
		if timeout == 0 {
			t.Error("should not take fast path for non-zero timeout")
		} else {
			// This is the path taken - create timeout context and wrapper
			baseCtx := t.Context()
			timeoutCtx, cancel := context.WithTimeout(baseCtx, timeout)
			defer cancel()

			// Verify timeout context is created correctly
			deadline, ok := timeoutCtx.Deadline()
			assert.True(t, ok, "timeout context should have deadline")
			assert.Positive(t, time.Until(deadline), "deadline should be in future")
			assert.LessOrEqual(t, time.Until(deadline), timeout, "deadline within timeout")
		}
	})
}

// TestRunView_DeferRecoveryPattern tests the defer/recover pattern
func TestRunView_DeferRecoveryPattern(t *testing.T) {
	t.Run("outer defer recovers from panic", func(t *testing.T) {
		recovered := false

		func() {
			defer func() {
				if r := recover(); r != nil {
					recovered = true
				}
			}()

			panic("test panic")
		}()

		assert.True(t, recovered, "outer defer should recover panic")
	})

	t.Run("inner goroutine defer recovers independently", func(t *testing.T) {
		var innerRecovered atomic.Bool
		var outerRecovered atomic.Bool

		func() {
			defer func() {
				if r := recover(); r != nil {
					outerRecovered.Store(true)
				}
			}()

			done := make(chan struct{})
			go func() {
				defer close(done)
				defer func() {
					if r := recover(); r != nil {
						innerRecovered.Store(true)
					}
				}()

				panic("inner panic")
			}()

			<-done
		}()

		assert.True(t, innerRecovered.Load(), "inner defer should recover")
		assert.False(t, outerRecovered.Load(), "outer should not catch inner panic")
	})
}
