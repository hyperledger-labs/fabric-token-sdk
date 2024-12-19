/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabric

import (
	"sync"
	"time"
)

type rwLock interface {
	Lock()
	Unlock()
	RLock()
	RUnlock()
}

type CacheMap[K comparable, V any] interface {
	Get(K) (V, bool)
	Put(K, V)
	Update(K, func(bool, V) (bool, V)) bool
	Delete(...K)
	Len() int
}

type noLock struct{}

func (l *noLock) Lock()    {}
func (l *noLock) Unlock()  {}
func (l *noLock) RLock()   {}
func (l *noLock) RUnlock() {}

// NewLRUCache creates a cache with limited size with LRU eviction policy
func NewLRUCache[K comparable, V any](size, buffer int, onEvict func(map[K]V)) *evictionCache[K, V] {
	m := map[K]V{}
	return &evictionCache[K, V]{
		m:              m,
		l:              &noLock{},
		evictionPolicy: NewLRUEviction(size, buffer, func(keys []K) { evict(keys, m, onEvict) }),
	}
}

func NewTimeoutCache[K comparable, V any](evictionTimeout time.Duration, onEvict func(map[K]V)) *evictionCache[K, V] {
	m := map[K]V{}
	l := &sync.RWMutex{}
	return &evictionCache[K, V]{
		m: m,
		l: l,
		evictionPolicy: NewTimeoutEviction(evictionTimeout, func(keys []K) {
			l.Lock()
			defer l.Unlock()
			evict(keys, m, onEvict)
		}),
	}
}

func evict[K comparable, V any](keys []K, m map[K]V, onEvict func(map[K]V)) {
	evicted := make(map[K]V, len(keys))
	for _, k := range keys {
		if v, ok := m[k]; ok {
			evicted[k] = v
			delete(m, k)
		} else {
			logger.Debugf("No need to evict [%k]. Was already deleted.")
		}
	}
	onEvict(evicted)
}

type evictionCache[K comparable, V any] struct {
	m              map[K]V
	l              rwLock
	evictionPolicy EvictionPolicy[K]
}

type EvictionPolicy[K comparable] interface {
	// Push adds a key and must be invoked under write-lock
	Push(K)
}

func (c *evictionCache[K, V]) Get(key K) (V, bool) {
	c.l.RLock()
	defer c.l.RUnlock()
	v, ok := c.m[key]
	return v, ok
}

func (c *evictionCache[K, V]) Put(key K, value V) {
	c.l.Lock()
	defer c.l.Unlock()
	c.m[key] = value
	// We assume that a value is always new for performance reasons.
	// If we try to put again a value, this value will be put also in the LRU keys instead of just promoting the existing one.
	// If we put this value c.cap times, then this will evict all other values.
	c.evictionPolicy.Push(key)
}

func (c *evictionCache[K, V]) Update(key K, f func(bool, V) (bool, V)) bool {
	c.l.Lock()
	defer c.l.Unlock()
	v, ok := c.m[key]
	keep, newValue := f(ok, v)
	if !keep {
		delete(c.m, key)
	} else {
		c.m[key] = newValue
	}
	if !ok && keep {
		c.evictionPolicy.Push(key)
	}
	return ok
}

func (c *evictionCache[K, V]) Delete(keys ...K) {
	c.l.Lock()
	defer c.l.Unlock()
	for _, key := range keys {
		delete(c.m, key)
	}
}

func (c *evictionCache[K, V]) Len() int {
	return len(c.m)
}

type timeoutEviction[K comparable] struct {
	keys  []timeoutEntry[K]
	mu    sync.RWMutex
	evict func([]K)
}

type timeoutEntry[K comparable] struct {
	created time.Time
	key     K
}

func NewTimeoutEviction[K comparable](timeout time.Duration, evict func([]K)) *timeoutEviction[K] {
	e := &timeoutEviction[K]{
		keys:  make([]timeoutEntry[K], 0),
		evict: evict,
	}
	go e.cleanup(timeout)
	return e
}

func (e *timeoutEviction[K]) cleanup(timeout time.Duration) {
	logger.Infof("Launch cleanup function with eviction timeout [%v]", timeout)
	for range time.Tick(1 * time.Second) {
		expiry := time.Now().Add(timeout)
		e.mu.RLock()
		evicted := make([]K, 0)
		for _, entry := range e.keys {
			if entry.created.Before(expiry) {
				break
			}
			evicted = append(evicted, entry.key)
		}
		e.mu.RUnlock()
		if len(evicted) > 0 {
			e.mu.Lock()
			e.keys = e.keys[len(evicted):]
			e.mu.Unlock()
			logger.Infof("Evicting %d entries", len(evicted))
			e.evict(evicted)
		}
	}
}

func (e *timeoutEviction[K]) Push(key K) {
	e.keys = append(e.keys, timeoutEntry[K]{key: key, created: time.Now()})
}

func NewLRUEviction[K comparable](size, buffer int, evict func([]K)) *lruEviction[K] {
	return &lruEviction[K]{
		size:  size,
		cap:   size + buffer,
		keys:  make([]K, 0, size+buffer),
		evict: evict,
	}
}

type lruEviction[K comparable] struct {
	// size is the minimum amount of entries guaranteed to be kept in cache.
	size int
	// cap + size is the maximum amount of entries that can be kept in cache. After that, a cleanup is invoked.
	cap int
	// keys keeps track of which keys should be evicted.
	// The last element of the slice is the most recent one.
	// Performance improvement: keep sliding index to avoid reallocating
	keys []K
	// evict is called when we evict
	evict func([]K)
	mu    sync.Mutex
}

func (c *lruEviction[K]) Push(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.keys = append(c.keys, key)
	if len(c.keys) <= c.cap {
		return
	}
	logger.Infof("Capacity of %d exceeded. Evicting old keys by shifting LRU keys keeping only the %d most recent ones", c.cap, c.size)
	c.evict(c.keys[0 : c.cap-c.size])
	c.keys = (c.keys)[c.cap-c.size:]
}

// NewMapCache creates a cache with unlimited size
func NewMapCache[K comparable, V any]() *mapCache[K, V] {
	return &mapCache[K, V]{
		m: map[K]V{},
		l: &noLock{},
	}
}

type mapCache[K comparable, V any] struct {
	m map[K]V
	l rwLock
}

func (c *mapCache[K, V]) Get(key K) (V, bool) {
	c.l.RLock()
	defer c.l.RUnlock()
	v, ok := c.m[key]
	return v, ok
}

func (c *mapCache[K, V]) Put(key K, value V) {
	c.l.Lock()
	defer c.l.Unlock()
	c.m[key] = value
}

func (c *mapCache[K, V]) Delete(keys ...K) {
	c.l.Lock()
	defer c.l.Unlock()
	for _, key := range keys {
		delete(c.m, key)
	}
}

func (c *mapCache[K, V]) Update(key K, f func(bool, V) (bool, V)) bool {
	c.l.Lock()
	defer c.l.Unlock()
	v, ok := c.m[key]
	keep, newValue := f(ok, v)
	if !keep {
		delete(c.m, key)
	} else {
		c.m[key] = newValue
	}
	return ok
}

func (c *mapCache[K, V]) Len() int {
	return len(c.m)
}
