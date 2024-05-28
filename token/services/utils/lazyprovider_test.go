/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package utils

import (
	"fmt"
	"sync"
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/pkg/errors"
)

type entry struct{ key, value string }

func TestNoErrors(t *testing.T) {
	cache := newTestCache()

	// Get non existing
	val, err := cache.Get(entry{"key", "v1"})
	assert.NoError(err)
	assert.Equal("v1", val)

	// Get existing
	val, err = cache.Get(entry{"key", "v2"})
	assert.NoError(err)
	assert.Equal("v1", val)

	// Update existing
	oldVal, newVal, err := cache.Update(entry{"key", "v3"})
	assert.NoError(err)
	assert.Equal("v1", oldVal)
	assert.Equal("v3", newVal)

	// Get updated
	val, err = cache.Get(entry{"key", "v4"})
	assert.NoError(err)
	assert.Equal("v3", val)

	// Delete existing
	val, ok := cache.Delete(entry{"key", "v5"})
	assert.True(ok)
	assert.Equal("v3", val)

	// Get deleted
	val, err = cache.Get(entry{"key", "v6"})
	assert.NoError(err)
	assert.Equal("v6", val)
}

func TestDeleteNonExisting(t *testing.T) {
	cache := newTestCache()

	val, ok := cache.Delete(entry{"key", "v1"})
	assert.False(ok)
	assert.Equal("", val)
}

func TestUpdateNonExisting(t *testing.T) {
	cache := newTestCache()

	oldVal, newVal, err := cache.Update(entry{"key", "v1"})
	assert.NoError(err)
	assert.Equal("", oldVal)
	assert.Equal("v1", newVal)
}

func TestError(t *testing.T) {
	cache := newTestCache()

	val, err := cache.Get(entry{"error", "e1"})
	assert.Error(err, "e1")
	assert.Equal("", val)

	val, err = cache.Get(entry{"key", "v1"})
	assert.NoError(err)
	assert.Equal("v1", val)
}

func TestParallel(t *testing.T) {
	const iterations = 100
	cache := newTestCache()
	vals := make(chan string, iterations)
	done := make(chan struct{})

	values := make(map[string]struct{})
	go func() {
		for v := range vals {
			values[v] = struct{}{}
		}
		done <- struct{}{}
	}()

	var wg sync.WaitGroup
	wg.Add(iterations)
	for i := 0; i < iterations; i++ {
		val := fmt.Sprintf("v%d", i)
		go func() {
			val, err := cache.Get(entry{"key", val})
			assert.NoError(err)
			vals <- val
			wg.Done()
		}()
	}
	wg.Wait()
	close(vals)

	<-done
	assert.Equal(1, cache.Length(), "we only updated one key")
	assert.Equal(1, len(values), "we always got one value back (the one we first set)")
}

func newTestCache() LazyProvider[entry, string] {
	return NewLazyProviderWithKeyMapper(func(in entry) string {
		return in.key
	}, func(in entry) (string, error) {
		if in.key == "error" {
			return "", errors.New(in.value)
		}
		return in.value, nil
	})
}
