/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package queue

import (
	"context"
	"errors"
	"log"
	"runtime/debug"
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

// EventQueue manages a pool of workers processing events
type EventQueue struct {
	workers      int
	queueSize    int
	events       chan Event
	wg           sync.WaitGroup
	ctx          context.Context
	cancel       context.CancelFunc
	shutdownOnce sync.Once
	closed       int32
}

// Config holds configuration for the EventQueue
type Config struct {
	Workers   int // Number of worker goroutines
	QueueSize int // Size of the event buffer
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
		workers:   cfg.Workers,
		queueSize: cfg.QueueSize,
		events:    make(chan Event, cfg.QueueSize),
		ctx:       ctx,
		cancel:    cancel,
		closed:    0,
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
			logger.Infof("Worker %d recovered from panic: %v, stack [%s]", id, r, string(debug.Stack()))
			// Restart the worker to maintain pool size
			eq.wg.Add(1)
			go eq.worker(id)
		}
	}()

	for {
		select {
		case event, ok := <-eq.events:
			if !ok {
				// Channel closed, worker exits
				logger.Infof("Worker %d shutting down", id)
				return
			}

			// Process the event with context
			if err := event.Process(eq.ctx); err != nil {
				logger.Infof("Worker %d: error processing event: %v", id, err)
			}

		case <-eq.ctx.Done():
			// Context cancelled, drain remaining events before exit
			logger.Infof("Worker %d received shutdown signal", id)
			return
		}
	}
}

// Enqueue adds an event to the queue (non-blocking)
func (eq *EventQueue) Enqueue(event Event) (err error) {
	if eq.isClosed() {
		return ErrQueueClosed
	}

	defer func() {
		if r := recover(); r != nil {
			err = ErrQueueClosed
		}
	}()

	select {
	case eq.events <- event:
		return nil
	default:
		return ErrQueueFull
	}
}

// EnqueueBlocking adds an event to the queue (blocks until space available or timeout)
func (eq *EventQueue) EnqueueBlocking(ctx context.Context, event Event) (err error) {
	if eq.isClosed() {
		return ErrQueueClosed
	}

	logger.Infof("EnqueueBlocking event: %v", event)

	defer func() {
		if r := recover(); r != nil {
			err = ErrQueueClosed
		}
	}()

	select {
	case eq.events <- event:
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
		atomic.StoreInt32(&eq.closed, 1)

		// Close the events channel to signal no more work
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
				log.Println("All workers shut down gracefully")
			case <-time.After(timeout):
				eq.cancel() // Force cancel remaining work
				shutdownErr = ErrShutdownTimeout
				log.Println("Shutdown timeout exceeded, forcing cancellation")
			}
		} else {
			<-done
			log.Println("All workers shut down gracefully")
		}

		eq.cancel()
	})

	return shutdownErr
}

// Stats returns queue statistics
func (eq *EventQueue) Stats() Stats {
	return Stats{
		Workers:   eq.workers,
		QueueSize: eq.queueSize,
		Pending:   len(eq.events),
		IsClosed:  eq.isClosed(),
	}
}

func (eq *EventQueue) isClosed() bool {
	return atomic.LoadInt32(&eq.closed) == 1
}
