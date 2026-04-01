/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

// This file tests holder.go which implements a generic driver registry.
// Tests cover registration, retrieval, concurrency, and duplicate detection.
package drivers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewHolder verifies NewHolder creates a holder with initialized map
func TestNewHolder(t *testing.T) {
	t.Run("string key, int value", func(t *testing.T) {
		holder := NewHolder[string, int]()

		assert.NotNil(t, holder)
		assert.NotNil(t, holder.Logger)
		assert.NotNil(t, holder.Drivers)
		assert.Empty(t, holder.Drivers)
	})

	t.Run("int key, string value", func(t *testing.T) {
		holder := NewHolder[int, string]()

		assert.NotNil(t, holder)
		assert.NotNil(t, holder.Drivers)
	})
}

// TestHolder_Register verifies Register adds drivers correctly
func TestHolder_Register(t *testing.T) {
	t.Run("register single driver", func(t *testing.T) {
		holder := NewHolder[string, *testDriver]()
		driver := &testDriver{name: "test"}

		holder.Register("driver1", driver)

		assert.Len(t, holder.Drivers, 1)
		retrieved, ok := holder.Get("driver1")
		require.True(t, ok)
		assert.Equal(t, driver, retrieved)
	})

	t.Run("register multiple drivers", func(t *testing.T) {
		holder := NewHolder[string, *testDriver]()
		driver1 := &testDriver{name: "driver1"}
		driver2 := &testDriver{name: "driver2"}

		holder.Register("d1", driver1)
		holder.Register("d2", driver2)

		assert.Len(t, holder.Drivers, 2)
	})
}

// TestHolder_Register_Panic verifies Register panics on invalid input
func TestHolder_Register_Panic(t *testing.T) {
	t.Run("panic on nil driver", func(t *testing.T) {
		holder := NewHolder[string, *testDriver]()

		assert.Panics(t, func() {
			holder.Register("nil-driver", nil)
		}, "should panic when registering nil driver")
	})

	t.Run("panic on duplicate registration", func(t *testing.T) {
		holder := NewHolder[string, *testDriver]()
		driver := &testDriver{name: "test"}

		holder.Register("driver1", driver)

		assert.Panics(t, func() {
			holder.Register("driver1", driver)
		}, "should panic when registering same name twice")
	})
}

// TestHolder_Get verifies Get retrieves registered drivers
func TestHolder_Get(t *testing.T) {
	t.Run("get existing driver", func(t *testing.T) {
		holder := NewHolder[string, *testDriver]()
		driver := &testDriver{name: "test"}
		holder.Register("driver1", driver)

		retrieved, ok := holder.Get("driver1")

		assert.True(t, ok)
		assert.Equal(t, driver, retrieved)
	})

	t.Run("get non-existing driver", func(t *testing.T) {
		holder := NewHolder[string, *testDriver]()

		retrieved, ok := holder.Get("non-existing")

		assert.False(t, ok)
		assert.Nil(t, retrieved)
	})

	t.Run("get after multiple registrations", func(t *testing.T) {
		holder := NewHolder[string, *testDriver]()
		driver1 := &testDriver{name: "driver1"}
		driver2 := &testDriver{name: "driver2"}
		driver3 := &testDriver{name: "driver3"}

		holder.Register("d1", driver1)
		holder.Register("d2", driver2)
		holder.Register("d3", driver3)

		// Verify each can be retrieved
		r1, ok1 := holder.Get("d1")
		assert.True(t, ok1)
		assert.Equal(t, driver1, r1)

		r2, ok2 := holder.Get("d2")
		assert.True(t, ok2)
		assert.Equal(t, driver2, r2)

		r3, ok3 := holder.Get("d3")
		assert.True(t, ok3)
		assert.Equal(t, driver3, r3)
	})
}

// TestHolder_ConcurrentAccess verifies thread-safe access
func TestHolder_ConcurrentAccess(t *testing.T) {
	holder := NewHolder[int, *testDriver]()

	// Register some initial drivers
	for i := range 10 {
		holder.Register(i, &testDriver{name: string(rune('A' + i))})
	}

	// Concurrent reads should not panic
	done := make(chan bool)
	for i := range 10 {
		go func(id int) {
			defer func() { done <- true }()
			for range 100 {
				_, _ = holder.Get(id % 10)
			}
		}(i)
	}

	// Wait for all goroutines
	for range 10 {
		<-done
	}
}

// TestHolder_DifferentTypes verifies holder works with different types
func TestHolder_DifferentTypes(t *testing.T) {
	t.Run("interface values", func(t *testing.T) {
		type Driver interface {
			Name() string
		}
		holder := NewHolder[string, Driver]()
		driver := &testDriver{name: "test"}

		holder.Register("driver1", driver)

		retrieved, ok := holder.Get("driver1")
		assert.True(t, ok)
		assert.Equal(t, "test", retrieved.Name())
	})

	t.Run("struct values", func(t *testing.T) {
		type Config struct {
			Value string
		}
		holder := NewHolder[string, Config]()
		config := Config{Value: "test"}

		// This should panic because struct is not nil-able
		assert.Panics(t, func() {
			holder.Register("config1", config)
		})
	})
}

// testDriver is a simple test driver implementation
type testDriver struct {
	name string
}

func (d *testDriver) Name() string {
	return d.name
}
