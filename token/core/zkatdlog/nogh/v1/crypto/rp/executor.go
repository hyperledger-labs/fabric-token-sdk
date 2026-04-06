/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rp

// Executor defines a minimal interface for submitting tasks and waiting for their completion.
type Executor interface {
	Submit(func())
	Wait()
}

// SerialExecutor executes tasks sequentially when Wait is invoked.
type SerialExecutor struct {
	tasks []func()
}

// NewSerialExecutor creates a new serial executor.
func NewSerialExecutor() *SerialExecutor {
	return &SerialExecutor{
		tasks: make([]func(), 0),
	}
}

// Submit stores the task for later execution.
func (s *SerialExecutor) Submit(task func()) {
	s.tasks = append(s.tasks, task)
}

// Wait executes all submitted tasks sequentially.
func (s *SerialExecutor) Wait() {
	for _, task := range s.tasks {
		task()
	}
	// reset tasks
	s.tasks = s.tasks[:0]
}
