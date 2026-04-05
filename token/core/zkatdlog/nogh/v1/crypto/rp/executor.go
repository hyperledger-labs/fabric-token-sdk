/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rp

import (
	"runtime"
	"sync"
)

// Executor defines a minimal interface for submitting tasks and waiting for their completion.
type Executor interface {
	Submit(func())
	Wait()
}

// SerialExecutor executes tasks immediately in the caller goroutine.
// This provides the lowest latency and zero scheduling overhead.
type SerialExecutor struct{}

func (s *SerialExecutor) Submit(task func()) {
	task()
}

func (s *SerialExecutor) Wait() {}

// WorkerPoolExecutor executes tasks using a fixed-size worker pool.
// It bounds concurrency and avoids spawning unbounded goroutines.
type WorkerPoolExecutor struct {
	tasks chan func()
	wg    sync.WaitGroup
}

// NewWorkerPoolExecutor creates a worker pool with numWorkers workers.
func NewWorkerPoolExecutor(numWorkers int) *WorkerPoolExecutor {
	e := &WorkerPoolExecutor{
		tasks: make(chan func()),
	}

	for i := 0; i < numWorkers; i++ {
		go func() {
			for task := range e.tasks {
				task()
				e.wg.Done()
			}
		}()
	}

	return e
}

// Submit adds a task to the worker pool.
func (e *WorkerPoolExecutor) Submit(task func()) {
	e.wg.Add(1)
	e.tasks <- task
}

// Wait blocks until all submitted tasks are completed.
func (e *WorkerPoolExecutor) Wait() {
	e.wg.Wait()
}

// Close shuts down the worker pool.
// Optional: call only if you want to clean up goroutines explicitly.
func (e *WorkerPoolExecutor) Close() {
	close(e.tasks)
}

// minParallelWork defines the minimum work size required to justify parallel execution. Below this threshold,
// serial execution is faster due to lower overhead.
const minParallelWork = 8

// NewExecutor creates an appropriate executor based on workload size.
// Small workloads -> SerialExecutor
// Large workloads -> WorkerPoolExecutor with bounded concurrency
func NewExecutor(workSize int) Executor {
	// Small workload: avoid parallel overhead
	if workSize < minParallelWork {
		return &SerialExecutor{}
	}

	procs := runtime.GOMAXPROCS(0)

	// Use ~75% of available CPUs to balance throughput and latency
	numWorkers := procs - procs/4
	if numWorkers < 1 {
		numWorkers = 1
	}

	// Do not create more workers than tasks
	if numWorkers > workSize {
		numWorkers = workSize
	}

	return NewWorkerPoolExecutor(numWorkers)
}
