/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSerialExecutorBasic(t *testing.T) {
	exec := NewSerialExecutor()

	result := 0

	exec.Submit(func() { result += 1 })
	exec.Submit(func() { result += 2 })

	exec.Wait()

	if result != 3 {
		t.Fatalf("expected 3, got %d", result)
	}
}

func TestSerialExecutorOrder(t *testing.T) {
	exec := NewSerialExecutor()

	var result []int

	exec.Submit(func() { result = append(result, 1) })
	exec.Submit(func() { result = append(result, 2) })
	exec.Submit(func() { result = append(result, 3) })

	exec.Wait()

	expected := []int{1, 2, 3}

	for i := range expected {
		if result[i] != expected[i] {
			t.Fatalf("expected %v, got %v", expected, result)
		}
	}
}

func TestSerialExecutorEmpty(t *testing.T) {
	exec := NewSerialExecutor()

	// should not panic
	exec.Wait()
}

func TestSerialExecutorReuse(t *testing.T) {
	exec := NewSerialExecutor()

	result := 0

	exec.Submit(func() { result += 1 })
	exec.Wait()

	exec.Submit(func() { result += 2 })
	exec.Wait()

	if result != 3 {
		t.Fatalf("expected 3, got %d", result)
	}
}

func TestSerialExecutorBatching(t *testing.T) {
	exec := NewSerialExecutor()

	counter := 0

	for range 5 {
		exec.Submit(func() { counter++ })
	}

	exec.Wait()

	if counter != 5 {
		t.Fatalf("expected 5, got %d", counter)
	}
}

func TestSerialExecutor_SingleTask(t *testing.T) {
	executor := NewSerialExecutor()
	called := false
	executor.Submit(func() { called = true })
	executor.Wait()
	assert.True(t, called)
}

func TestSerialExecutor_TasksMutateSharedSlice(t *testing.T) {
	// Mirrors the real usage in reduceGenerators and Prove():
	// tasks write to disjoint indices of a shared slice.
	executor := NewSerialExecutor()
	n := 64
	output := make([]int, n)
	for i := range n {
		executor.Submit(func() {
			output[i] = i + 1
		})
	}
	executor.Wait()
	for i := range n {
		assert.Equal(t, i+1, output[i], "index %d", i)
	}
}
