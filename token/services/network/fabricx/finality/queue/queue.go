/*

Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0

*/

package queue

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
)

var logger = logging.MustGetLogger()

var (
	// ErrQueueClosed is returned when an event is added to a closed queue
	ErrQueueClosed = errors.New("queue is closed")
	// ErrQueueFull is returned when a non-blocking enqueue fails because the queue is full
	ErrQueueFull = errors.New("queue is full")
	// ErrShutdownTimeout is returned when the shutdown timeout is exceeded
	ErrShutdownTimeout = errors.New("shutdown timeout exceeded")
)

// Event represents a unit of work to be processed
type Event interface {
	Process(ctx context.Context) error
}

// Stats represents statistics about the event queue
type Stats struct {
	// Workers is the number of worker goroutines
	Workers int
	// QueueSize is the size of the event buffer
	QueueSize int
	// Pending is the number of pending events in the queue
	Pending int
	// IsClosed is true if the queue is closed
	IsClosed bool
}

// Config holds configuration for the EventQueue
type Config struct {
	Workers   int // Number of worker goroutines
	QueueSize int // Size of the event buffer
}

// EventQueue manages a pool of workers processing events
type EventQueue struct {
	workers      int
	events       chan Event
	wg           sync.WaitGroup
	ctx          context.Context
	cancel       context.CancelFunc
	shutdownOnce sync.Once
	closed       bool
	mu           sync.RWMutex
}

// NewEventQueue creates and starts a new event queue with fixed workers
func NewEventQueue(cfg Config) (*EventQueue, error) {
	if cfg.Workers <= 0 {
		return nil, errors.New("workers must be greater than 0")
	}

	if cfg.QueueSize <= 0 {
		return nil, errors.New("queue size must be greater than 0")
	}

	ctx, cancel := context.WithCancel(context.Background())
	eq := &EventQueue{
		workers: cfg.Workers,
		events:  make(chan Event, cfg.QueueSize),
		ctx:     ctx,
		cancel:  cancel,
		closed:  false,
	}

	// Start worker pool
	eq.start()

	return eq, nil
}

// start initializes all worker goroutines
func (eq *EventQueue) start() {
	for i := range eq.workers {
		eq.wg.Add(1)
		go eq.worker(i)
	}
}

// worker processes events from the queue
func (eq *EventQueue) worker(id int) {
	defer eq.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("Worker %d recovered from panic: %v", id, r)
			// Don't restart worker to prevent unbounded goroutine creation
			// The pool will continue with remaining workers
		}
	}()

	for {
		select {
		case event, ok := <-eq.events:
			if !ok {
				// Channel closed, worker exits
				logger.Debugf("Worker %d shutting down", id)

				return
			}

			// Process the event with context
			if err := event.Process(eq.ctx); err != nil {
				logger.Debugf("Worker %d: error processing event: %v", id, err)
			}

		case <-eq.ctx.Done():
			// Context canceled, exit gracefully
			logger.Debugf("Worker %d received shutdown signal", id)

			return
		}
	}
}

// Enqueue adds an event to the queue (non-blocking)
func (eq *EventQueue) Enqueue(event Event) error {
	eq.mu.RLock()
	defer eq.mu.RUnlock()

	if eq.closed {
		return ErrQueueClosed
	}

	select {
	case eq.events <- event:
		return nil
	default:
		return ErrQueueFull
	}
}

// EnqueueBlocking adds an event to the queue (blocks until space available or timeout)
func (eq *EventQueue) EnqueueBlocking(ctx context.Context, event Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	eq.mu.RLock()
	defer eq.mu.RUnlock()

	if eq.closed {
		return ErrQueueClosed
	}

	select {
	case eq.events <- event:
		logger.Debugf("EnqueueBlocking event: [%v]", event)

		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-eq.ctx.Done():
		return ErrQueueClosed
	}
}

// Shutdown gracefully stops the queue with a timeout
func (eq *EventQueue) Shutdown(timeout time.Duration) error {
	var shutdownErr error
	eq.shutdownOnce.Do(func() {
		// Set closed flag first to prevent new enqueues
		eq.mu.Lock()
		eq.closed = true
		eq.mu.Unlock()

		// Cancel context to signal EnqueueBlocking calls to stop waiting
		eq.cancel()

		// Wait for all active senders to finish their select
		eq.mu.Lock()
		defer eq.mu.Unlock()

		// Close channel now that we're sure no one is in the select or will enter it
		close(eq.events)

		// Wait for workers to finish with timeout
		done := make(chan struct{})
		go func() {
			eq.wg.Wait()
			close(done)
		}()

		if timeout > 0 {
			select {
			case <-done:
				logger.Info("All workers shut down gracefully")
			case <-time.After(timeout):
				shutdownErr = ErrShutdownTimeout
				logger.Warnf("Shutdown timeout exceeded")
			}
		} else {
			<-done
			logger.Info("All workers shut down gracefully")
		}
	})

	return shutdownErr
}

// Stats returns queue statistics
func (eq *EventQueue) Stats() Stats {
	return Stats{
		Workers:   eq.workers,
		QueueSize: cap(eq.events),
		Pending:   len(eq.events),
		IsClosed:  eq.isClosed(),
	}
}

func (eq *EventQueue) isClosed() bool {
	eq.mu.RLock()
	defer eq.mu.RUnlock()

	return eq.closed
}
