/*

Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0

*/

package queue

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
)

var logger = logging.MustGetLogger()

var (
	ErrQueueClosed     = errors.New("queue is closed")
	ErrQueueFull       = errors.New("queue is full")
	ErrShutdownTimeout = errors.New("shutdown timeout exceeded")
)

// Event represents a unit of work to be processed
type Event interface {
	Process(ctx context.Context) error
}

type Stats struct {
	Workers   int
	QueueSize int
	Pending   int
	IsClosed  bool
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
	closed       int32
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
		closed:  0,
	}

	// Start worker pool
	eq.start()
	return eq, nil
}

// start initializes all worker goroutines
func (eq *EventQueue) start() {
	for i := 0; i < eq.workers; i++ {
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
	if eq.isClosed() {
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
	if eq.isClosed() {
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
		// Close channel first to prevent new enqueues
		close(eq.events)
		// Set closed flag
		atomic.StoreInt32(&eq.closed, 1)
		// Cancel context to signal workers
		eq.cancel()

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
	return atomic.LoadInt32(&eq.closed) == 1
}
