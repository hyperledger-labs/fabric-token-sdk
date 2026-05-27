/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultDBTimeoutConfig(t *testing.T) {
	config := DefaultDBTimeoutConfig()
	
	assert.Equal(t, 5*time.Second, config.ShortOpTimeout, "Short operation timeout should be 5 seconds")
	assert.Equal(t, 15*time.Second, config.MediumOpTimeout, "Medium operation timeout should be 15 seconds")
	assert.Equal(t, 30*time.Second, config.LongOpTimeout, "Long operation timeout should be 30 seconds")
}

func TestWithShortTimeout(t *testing.T) {
	ctx := context.Background()
	
	timeoutCtx, cancel := WithShortTimeout(ctx, nil)
	defer cancel()
	
	deadline, ok := timeoutCtx.Deadline()
	require.True(t, ok, "Context should have a deadline")
	
	// Verify the deadline is approximately 5 seconds from now
	expectedDeadline := time.Now().Add(5 * time.Second)
	assert.WithinDuration(t, expectedDeadline, deadline, 100*time.Millisecond, "Deadline should be approximately 5 seconds from now")
}

func TestWithMediumTimeout(t *testing.T) {
	ctx := context.Background()
	
	timeoutCtx, cancel := WithMediumTimeout(ctx, nil)
	defer cancel()
	
	deadline, ok := timeoutCtx.Deadline()
	require.True(t, ok, "Context should have a deadline")
	
	// Verify the deadline is approximately 15 seconds from now
	expectedDeadline := time.Now().Add(15 * time.Second)
	assert.WithinDuration(t, expectedDeadline, deadline, 100*time.Millisecond, "Deadline should be approximately 15 seconds from now")
}

func TestWithLongTimeout(t *testing.T) {
	ctx := context.Background()
	
	timeoutCtx, cancel := WithLongTimeout(ctx, nil)
	defer cancel()
	
	deadline, ok := timeoutCtx.Deadline()
	require.True(t, ok, "Context should have a deadline")
	
	// Verify the deadline is approximately 30 seconds from now
	expectedDeadline := time.Now().Add(30 * time.Second)
	assert.WithinDuration(t, expectedDeadline, deadline, 100*time.Millisecond, "Deadline should be approximately 30 seconds from now")
}

func TestWithCustomTimeout(t *testing.T) {
	ctx := context.Background()
	customDuration := 42 * time.Second
	
	timeoutCtx, cancel := WithCustomTimeout(ctx, customDuration)
	defer cancel()
	
	deadline, ok := timeoutCtx.Deadline()
	require.True(t, ok, "Context should have a deadline")
	
	// Verify the deadline is approximately 42 seconds from now
	expectedDeadline := time.Now().Add(customDuration)
	assert.WithinDuration(t, expectedDeadline, deadline, 100*time.Millisecond, "Deadline should be approximately 42 seconds from now")
}

func TestTimeoutContextCancellation(t *testing.T) {
	ctx := context.Background()
	
	timeoutCtx, cancel := WithShortTimeout(ctx, nil)
	
	// Cancel immediately
	cancel()
	
	// Context should be cancelled
	select {
	case <-timeoutCtx.Done():
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Context should have been cancelled immediately")
	}
	
	assert.Error(t, timeoutCtx.Err(), "Context error should be set")
}

func TestCustomConfig(t *testing.T) {
	customConfig := &DBTimeoutConfig{
		ShortOpTimeout:  1 * time.Second,
		MediumOpTimeout: 2 * time.Second,
		LongOpTimeout:   3 * time.Second,
	}
	
	ctx := context.Background()
	
	t.Run("Short timeout with custom config", func(t *testing.T) {
		timeoutCtx, cancel := WithShortTimeout(ctx, customConfig)
		defer cancel()
		
		deadline, ok := timeoutCtx.Deadline()
		require.True(t, ok)
		
		expectedDeadline := time.Now().Add(1 * time.Second)
		assert.WithinDuration(t, expectedDeadline, deadline, 100*time.Millisecond)
	})
	
	t.Run("Medium timeout with custom config", func(t *testing.T) {
		timeoutCtx, cancel := WithMediumTimeout(ctx, customConfig)
		defer cancel()
		
		deadline, ok := timeoutCtx.Deadline()
		require.True(t, ok)
		
		expectedDeadline := time.Now().Add(2 * time.Second)
		assert.WithinDuration(t, expectedDeadline, deadline, 100*time.Millisecond)
	})
	
	t.Run("Long timeout with custom config", func(t *testing.T) {
		timeoutCtx, cancel := WithLongTimeout(ctx, customConfig)
		defer cancel()
		
		deadline, ok := timeoutCtx.Deadline()
		require.True(t, ok)
		
		expectedDeadline := time.Now().Add(3 * time.Second)
		assert.WithinDuration(t, expectedDeadline, deadline, 100*time.Millisecond)
	})
}

func TestContextInheritance(t *testing.T) {
	// Create a parent context with a value
	type contextKey string
	key := contextKey("test-key")
	parentCtx := context.WithValue(context.Background(), key, "test-value")
	
	// Create a timeout context from the parent
	timeoutCtx, cancel := WithShortTimeout(parentCtx, nil)
	defer cancel()
	
	// Verify the value is inherited
	value := timeoutCtx.Value(key)
	assert.Equal(t, "test-value", value, "Context value should be inherited from parent")
	
	// Verify deadline is set
	_, ok := timeoutCtx.Deadline()
	assert.True(t, ok, "Timeout context should have a deadline")
}

// Made with Bob
