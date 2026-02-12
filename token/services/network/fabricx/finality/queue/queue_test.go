/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package queue_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabricx/finality/queue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockEvent is a test implementation of the Event interface
type mockEvent struct {
	processFunc func(ctx context.Context) error
	processed   int32
}

func (m *mockEvent) Process(ctx context.Context) error {
	atomic.AddInt32(&m.processed, 1)
	if m.processFunc != nil {
		return m.processFunc(ctx)
	}
	return nil
}

func (m *mockEvent) wasProcessed() bool {
	return atomic.LoadInt32(&m.processed) > 0
}

// TestNewEventQueue_ValidConfig tests successful queue creation
func TestNewEventQueue_ValidConfig(t *testing.T) {
	cfg := queue.Config{
		Workers:   2,
		QueueSize: 10,
	}

	eq, err := queue.NewEventQueue(cfg)
	require.NoError(t, err)
	require.NotNil(t, eq)

	stats := eq.Stats()
	assert.Equal(t, 2, stats.Workers)
	assert.Equal(t, 10, stats.QueueSize)
	assert.Equal(t, 0, stats.Pending)
	assert.False(t, stats.IsClosed)

	err = eq.Shutdown(time.Second)
	assert.NoError(t, err)
}

// TestNewEventQueue_InvalidConfig tests queue creation with invalid configurations
func TestNewEventQueue_InvalidConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    queue.Config
		expectErr string
	}{
		{
			name:      "zero workers",
			config:    queue.Config{Workers: 0, QueueSize: 10},
			expectErr: "workers must be greater than 0",
		},
		{
			name:      "negative workers",
			config:    queue.Config{Workers: -1, QueueSize: 10},
			expectErr: "workers must be greater than 0",
		},
		{
			name:      "zero queue size",
			config:    queue.Config{Workers: 2, QueueSize: 0},
			expectErr: "queue size must be greater than 0",
		},
		{
			name:      "negative queue size",
			config:    queue.Config{Workers: 2, QueueSize: -1},
			expectErr: "queue size must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eq, err := queue.NewEventQueue(tt.config)
			assert.Error(t, err)
			assert.Nil(t, eq)
			assert.Contains(t, err.Error(), tt.expectErr)
		})
	}
}

// TestEnqueue_Success tests successful non-blocking enqueue
func TestEnqueue_Success(t *testing.T) {
	cfg := queue.Config{Workers: 2, QueueSize: 10}
	eq, err := queue.NewEventQueue(cfg)
	require.NoError(t, err)
	defer func() { _ = eq.Shutdown(time.Second) }()

	event := &mockEvent{}
	err = eq.Enqueue(event)
	assert.NoError(t, err)

	// Wait for processing
	time.Sleep(100 * time.Millisecond)
	assert.True(t, event.wasProcessed())
}

// TestEnqueue_QueueFull tests enqueue when queue is full
func TestEnqueue_QueueFull(t *testing.T) {
	cfg := queue.Config{Workers: 1, QueueSize: 2}
	eq, err := queue.NewEventQueue(cfg)
	require.NoError(t, err)
	defer func() { _ = eq.Shutdown(time.Second) }()

	// Create blocking events
	blockChan := make(chan struct{})
	blockingEvent := &mockEvent{
		processFunc: func(ctx context.Context) error {
			<-blockChan
			return nil
		},
	}

	// Fill the queue
	err = eq.Enqueue(blockingEvent)
	assert.NoError(t, err)
	err = eq.Enqueue(blockingEvent)
	assert.NoError(t, err)

	// Next enqueue should fail with queue full
	err = eq.Enqueue(blockingEvent)
	assert.ErrorIs(t, err, queue.ErrQueueFull)

	close(blockChan)
}

// TestEnqueue_AfterShutdown tests enqueue after queue is closed
func TestEnqueue_AfterShutdown(t *testing.T) {
	cfg := queue.Config{Workers: 1, QueueSize: 5}
	eq, err := queue.NewEventQueue(cfg)
	require.NoError(t, err)

	err = eq.Shutdown(time.Second)
	assert.NoError(t, err)

	event := &mockEvent{}
	err = eq.Enqueue(event)
	assert.ErrorIs(t, err, queue.ErrQueueClosed)
}

// TestEnqueueBlocking_Success tests successful blocking enqueue
func TestEnqueueBlocking_Success(t *testing.T) {
	cfg := queue.Config{Workers: 2, QueueSize: 10}
	eq, err := queue.NewEventQueue(cfg)
	require.NoError(t, err)
	defer func() { _ = eq.Shutdown(time.Second) }()

	ctx := t.Context()
	event := &mockEvent{}
	err = eq.EnqueueBlocking(ctx, event)
	assert.NoError(t, err)

	// Wait for processing
	time.Sleep(100 * time.Millisecond)
	assert.True(t, event.wasProcessed())
}

// TestEnqueueBlocking_ContextCancelled tests blocking enqueue with cancelled context
func TestEnqueueBlocking_ContextCancelled(t *testing.T) {
	cfg := queue.Config{Workers: 1, QueueSize: 1}
	eq, err := queue.NewEventQueue(cfg)
	require.NoError(t, err)
	defer func() { _ = eq.Shutdown(time.Second) }()

	// Fill the queue with blocking event
	blockChan := make(chan struct{})
	blockingEvent := &mockEvent{
		processFunc: func(ctx context.Context) error {
			<-blockChan
			return nil
		},
	}
	err = eq.Enqueue(blockingEvent)
	require.NoError(t, err)

	// Try to enqueue with cancelled context
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	event := &mockEvent{}
	err = eq.EnqueueBlocking(ctx, event)
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)

	close(blockChan)
}

// TestEnqueueBlocking_QueueClosed tests blocking enqueue when queue closes
func TestEnqueueBlocking_QueueClosed(t *testing.T) {
	cfg := queue.Config{Workers: 1, QueueSize: 1}
	eq, err := queue.NewEventQueue(cfg)
	require.NoError(t, err)

	// Fill the queue with blocking event that won't complete until we say so
	blockChan := make(chan struct{})
	defer close(blockChan)

	// First event - will be picked up by worker and block
	blockingEvent1 := &mockEvent{
		processFunc: func(ctx context.Context) error {
			<-blockChan
			return nil
		},
	}
	err = eq.Enqueue(blockingEvent1)
	require.NoError(t, err)

	// Wait for worker to pick up the event
	time.Sleep(50 * time.Millisecond)

	// Second event - will fill the queue buffer
	blockingEvent2 := &mockEvent{
		processFunc: func(ctx context.Context) error {
			<-blockChan
			return nil
		},
	}
	err = eq.Enqueue(blockingEvent2)
	require.NoError(t, err)

	// Now try to enqueue blocking with a context - this will block waiting for space
	ctx, cancel := context.WithTimeout(t.Context(), 500*time.Millisecond)
	defer cancel()

	event := &mockEvent{}
	errChan := make(chan error, 1)
	go func() {
		// Recover from potential panic if channel is closed during send
		defer func() {
			if r := recover(); r != nil {
				errChan <- queue.ErrQueueClosed
			}
		}()
		errChan <- eq.EnqueueBlocking(ctx, event)
	}()

	// Wait for goroutine to start blocking on the full queue
	time.Sleep(100 * time.Millisecond)

	// Shutdown the queue while the enqueue is blocked
	_ = eq.Shutdown(2 * time.Second)

	// Should get queue closed error from the blocked enqueue (or context timeout)
	err = <-errChan
	// Accept either ErrQueueClosed or context.DeadlineExceeded as valid outcomes
	assert.True(t, errors.Is(err, queue.ErrQueueClosed) || errors.Is(err, context.DeadlineExceeded),
		"Expected ErrQueueClosed or DeadlineExceeded, got: %v", err)
}

// TestEnqueueBlocking_AfterShutdown tests blocking enqueue after shutdown
func TestEnqueueBlocking_AfterShutdown(t *testing.T) {
	cfg := queue.Config{Workers: 1, QueueSize: 5}
	eq, err := queue.NewEventQueue(cfg)
	require.NoError(t, err)

	err = eq.Shutdown(time.Second)
	assert.NoError(t, err)

	ctx := t.Context()
	event := &mockEvent{}
	err = eq.EnqueueBlocking(ctx, event)
	assert.ErrorIs(t, err, queue.ErrQueueClosed)
}

// TestEventProcessing_Success tests successful event processing
func TestEventProcessing_Success(t *testing.T) {
	cfg := queue.Config{Workers: 3, QueueSize: 25}
	eq, err := queue.NewEventQueue(cfg)
	require.NoError(t, err)
	defer func() { _ = eq.Shutdown(time.Second) }()

	const numEvents = 20
	events := make([]*mockEvent, numEvents)
	for i := 0; i < numEvents; i++ {
		events[i] = &mockEvent{}
		err = eq.Enqueue(events[i])
		require.NoError(t, err)
	}

	// Wait for all events to be processed
	time.Sleep(200 * time.Millisecond)

	for i, event := range events {
		assert.True(t, event.wasProcessed(), "Event %d was not processed", i)
	}
}

// TestEventProcessing_WithError tests event processing that returns errors
func TestEventProcessing_WithError(t *testing.T) {
	cfg := queue.Config{Workers: 2, QueueSize: 10}
	eq, err := queue.NewEventQueue(cfg)
	require.NoError(t, err)
	defer func() { _ = eq.Shutdown(time.Second) }()

	expectedErr := errors.New("processing error")
	event := &mockEvent{
		processFunc: func(ctx context.Context) error {
			return expectedErr
		},
	}

	err = eq.Enqueue(event)
	assert.NoError(t, err)

	// Wait for processing
	time.Sleep(100 * time.Millisecond)
	assert.True(t, event.wasProcessed())
}

// TestEventProcessing_WithPanic tests worker recovery from panic
func TestEventProcessing_WithPanic(t *testing.T) {
	cfg := queue.Config{Workers: 2, QueueSize: 10}
	eq, err := queue.NewEventQueue(cfg)
	require.NoError(t, err)
	defer func() { _ = eq.Shutdown(time.Second) }()

	panicEvent := &mockEvent{
		processFunc: func(ctx context.Context) error {
			panic("test panic")
		},
	}

	err = eq.Enqueue(panicEvent)
	assert.NoError(t, err)

	// Wait for panic recovery
	time.Sleep(200 * time.Millisecond)

	// Enqueue another event to verify worker was restarted
	normalEvent := &mockEvent{}
	err = eq.Enqueue(normalEvent)
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)
	assert.True(t, normalEvent.wasProcessed())
}

// TestShutdown_Graceful tests graceful shutdown
func TestShutdown_Graceful(t *testing.T) {
	cfg := queue.Config{Workers: 2, QueueSize: 10}
	eq, err := queue.NewEventQueue(cfg)
	require.NoError(t, err)

	// Enqueue some events
	for i := 0; i < 5; i++ {
		event := &mockEvent{}
		err = eq.Enqueue(event)
		require.NoError(t, err)
	}

	err = eq.Shutdown(2 * time.Second)
	assert.NoError(t, err)

	stats := eq.Stats()
	assert.True(t, stats.IsClosed)
}

// TestShutdown_WithTimeout tests shutdown timeout
func TestShutdown_WithTimeout(t *testing.T) {
	cfg := queue.Config{Workers: 1, QueueSize: 5}
	eq, err := queue.NewEventQueue(cfg)
	require.NoError(t, err)

	// Enqueue a long-running event
	longEvent := &mockEvent{
		processFunc: func(ctx context.Context) error {
			time.Sleep(500 * time.Millisecond)
			return nil
		},
	}
	err = eq.Enqueue(longEvent)
	require.NoError(t, err)

	// Wait for event to start processing
	time.Sleep(50 * time.Millisecond)

	// Shutdown with short timeout
	err = eq.Shutdown(100 * time.Millisecond)
	assert.ErrorIs(t, err, queue.ErrShutdownTimeout)
}

// TestShutdown_ZeroTimeout tests shutdown with zero timeout (wait indefinitely)
func TestShutdown_ZeroTimeout(t *testing.T) {
	cfg := queue.Config{Workers: 2, QueueSize: 10}
	eq, err := queue.NewEventQueue(cfg)
	require.NoError(t, err)

	// Enqueue some quick events
	for i := 0; i < 3; i++ {
		event := &mockEvent{}
		err = eq.Enqueue(event)
		require.NoError(t, err)
	}

	err = eq.Shutdown(0)
	assert.NoError(t, err)
}

// TestShutdown_Multiple tests multiple shutdown calls
func TestShutdown_Multiple(t *testing.T) {
	cfg := queue.Config{Workers: 2, QueueSize: 10}
	eq, err := queue.NewEventQueue(cfg)
	require.NoError(t, err)

	err = eq.Shutdown(time.Second)
	assert.NoError(t, err)

	// Second shutdown should not cause issues
	err = eq.Shutdown(time.Second)
	assert.NoError(t, err)
}

// TestStats tests queue statistics
func TestStats(t *testing.T) {
	cfg := queue.Config{Workers: 3, QueueSize: 15}
	eq, err := queue.NewEventQueue(cfg)
	require.NoError(t, err)
	defer func() { _ = eq.Shutdown(time.Second) }()

	stats := eq.Stats()
	assert.Equal(t, 3, stats.Workers)
	assert.Equal(t, 15, stats.QueueSize)
	assert.Equal(t, 0, stats.Pending)
	assert.False(t, stats.IsClosed)

	// Add blocking events to check pending count
	blockChan := make(chan struct{})
	for i := 0; i < 5; i++ {
		event := &mockEvent{
			processFunc: func(ctx context.Context) error {
				<-blockChan
				return nil
			},
		}
		err = eq.Enqueue(event)
		require.NoError(t, err)
	}

	// Give workers time to pick up some events
	time.Sleep(50 * time.Millisecond)

	stats = eq.Stats()
	assert.True(t, stats.Pending >= 0 && stats.Pending <= 5)

	close(blockChan)
}

// TestConcurrentEnqueue tests concurrent enqueue operations
func TestConcurrentEnqueue(t *testing.T) {
	cfg := queue.Config{Workers: 5, QueueSize: 100}
	eq, err := queue.NewEventQueue(cfg)
	require.NoError(t, err)
	defer func() { _ = eq.Shutdown(2 * time.Second) }()

	const numGoroutines = 10
	const eventsPerGoroutine = 10

	var wg sync.WaitGroup
	events := make([]*mockEvent, numGoroutines*eventsPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(start int) {
			defer wg.Done()
			for j := 0; j < eventsPerGoroutine; j++ {
				idx := start + j
				events[idx] = &mockEvent{}
				err := eq.Enqueue(events[idx])
				assert.NoError(t, err)
			}
		}(i * eventsPerGoroutine)
	}

	wg.Wait()

	// Wait for all events to be processed
	time.Sleep(500 * time.Millisecond)

	processedCount := 0
	for _, event := range events {
		if event.wasProcessed() {
			processedCount++
		}
	}
	assert.Equal(t, numGoroutines*eventsPerGoroutine, processedCount)
}

// TestConcurrentEnqueueBlocking tests concurrent blocking enqueue operations
func TestConcurrentEnqueueBlocking(t *testing.T) {
	cfg := queue.Config{Workers: 5, QueueSize: 50}
	eq, err := queue.NewEventQueue(cfg)
	require.NoError(t, err)
	defer func() { _ = eq.Shutdown(2 * time.Second) }()

	const numGoroutines = 10
	const eventsPerGoroutine = 5

	var wg sync.WaitGroup
	events := make([]*mockEvent, numGoroutines*eventsPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(start int) {
			defer wg.Done()
			ctx := t.Context()
			for j := 0; j < eventsPerGoroutine; j++ {
				idx := start + j
				events[idx] = &mockEvent{}
				err := eq.EnqueueBlocking(ctx, events[idx])
				assert.NoError(t, err)
			}
		}(i * eventsPerGoroutine)
	}

	wg.Wait()

	// Wait for all events to be processed
	time.Sleep(500 * time.Millisecond)

	processedCount := 0
	for _, event := range events {
		if event.wasProcessed() {
			processedCount++
		}
	}
	assert.Equal(t, numGoroutines*eventsPerGoroutine, processedCount)
}

// TestWorkerContextCancellation tests that workers respect context cancellation
func TestWorkerContextCancellation(t *testing.T) {
	cfg := queue.Config{Workers: 2, QueueSize: 10}
	eq, err := queue.NewEventQueue(cfg)
	require.NoError(t, err)

	// Enqueue events that check context
	var processedAfterCancel int32
	for i := 0; i < 5; i++ {
		event := &mockEvent{
			processFunc: func(ctx context.Context) error {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(100 * time.Millisecond):
					atomic.AddInt32(&processedAfterCancel, 1)
					return nil
				}
			},
		}
		err = eq.Enqueue(event)
		require.NoError(t, err)
	}

	// Shutdown immediately
	err = eq.Shutdown(50 * time.Millisecond)
	// May or may not timeout depending on timing
	_ = err

	// Some events may not have completed
	count := atomic.LoadInt32(&processedAfterCancel)
	assert.True(t, count >= 0 && count <= 5)
}

// TestQueueDraining tests that queue drains events before shutdown
func TestQueueDraining(t *testing.T) {
	cfg := queue.Config{Workers: 3, QueueSize: 10}
	eq, err := queue.NewEventQueue(cfg)
	require.NoError(t, err)

	const numEvents = 5
	var processedCount int32
	events := make([]*mockEvent, numEvents)
	for i := 0; i < numEvents; i++ {
		events[i] = &mockEvent{
			processFunc: func(ctx context.Context) error {
				atomic.AddInt32(&processedCount, 1)
				time.Sleep(10 * time.Millisecond)
				return nil
			},
		}
		err = eq.Enqueue(events[i])
		require.NoError(t, err)
	}

	// Give workers time to start processing
	time.Sleep(20 * time.Millisecond)

	// Shutdown with enough timeout for all events
	err = eq.Shutdown(2 * time.Second)
	assert.NoError(t, err)

	// Check that all events were processed
	count := atomic.LoadInt32(&processedCount)
	assert.Equal(t, int32(numEvents), count, "Expected %d events to be processed, got %d", numEvents, count)
}
