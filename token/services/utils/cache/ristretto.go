/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cache

import (
	"github.com/dgraph-io/ristretto/v2"
	"golang.org/x/sync/singleflight"
)

const (
	// ZeroCost with this ristretto uses the Cost function defined in its configuration
	ZeroCost = 0
)

// ristrettoCache is our implementation using Ristretto v2.
type ristrettoCache[T any] struct {
	cache *ristretto.Cache[string, T]
	sfg   singleflight.Group
}

// NewRistrettoCache creates and returns a new ristretto-based cache implementation.
func NewRistrettoCache[T any](config *ristretto.Config[string, T]) (*ristrettoCache[T], error) {
	rCache, err := ristretto.NewCache[string, T](config)
	if err != nil {
		return nil, err
	}
	return &ristrettoCache[T]{
		cache: rCache,
	}, nil
}

func NewDefaultRistrettoCache[T any]() (*ristrettoCache[T], error) {
	return NewRistrettoCache[T](&ristretto.Config[string, T]{
		NumCounters: 1e6, // 1 million
		MaxCost:     1e8, // 100 million
		BufferItems: 64,
		Cost: func(value T) int64 {
			return 1
		},
	})
}

func (c *ristrettoCache[T]) Get(key string) (T, bool) {
	return c.cache.Get(key)
}

func (c *ristrettoCache[T]) Add(key string, value T) {
	c.cache.Set(key, value, ZeroCost)
	c.cache.Wait()
}

func (c *ristrettoCache[T]) Delete(key string) {
	c.cache.Del(key)
	c.cache.Wait()
}

func (c *ristrettoCache[T]) GetOrLoad(key string, loader func() (T, error)) (T, bool, error) {
	var zero T

	if value, found := c.Get(key); found {
		return value, true, nil
	}

	// If not found, use singleflight to prevent thundering herds.
	res, err, _ := c.sfg.Do(key, func() (interface{}, error) {
		newValue, loadErr := loader()
		if loadErr != nil {
			return nil, loadErr
		}
		c.Add(key, newValue)
		return newValue, nil
	})
	if err != nil {
		return zero, false, err
	}
	return res.(T), false, nil
}
