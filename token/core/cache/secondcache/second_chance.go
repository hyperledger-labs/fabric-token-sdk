/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package secondcache

import (
	"sync"
	"sync/atomic"

	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

// This package implements Second-Chance Algorithm, an approximate LRU algorithms.
// https://www.cs.jhu.edu/~yairamir/cs418/os6/tsld023.htm

// secondChanceCache holds key-value items with a limited size.
// When the number cached items exceeds the limit, victims are selected based on the
// Second-Chance Algorithm and Get purged
type secondChanceCache struct {
	// manages mapping between keys and items
	table map[token.ID]*cacheItem

	// holds a list of cached items.
	items []*cacheItem

	// indicates the next candidate of a victim in the items list
	position int

	// read lock for Get, and write lock for Add
	rwlock sync.RWMutex
}

type cacheItem struct {
	key   token.ID
	value interface{}
	// set to 1 when Get() is called. set to 0 when victim scan
	referenced int32
}

func New(cacheSize int) *secondChanceCache {
	var cache secondChanceCache
	cache.position = 0
	cache.items = make([]*cacheItem, cacheSize)
	cache.table = make(map[token.ID]*cacheItem)

	return &cache
}

func (cache *secondChanceCache) len() int {
	cache.rwlock.RLock()
	defer cache.rwlock.RUnlock()

	return len(cache.table)
}

func (cache *secondChanceCache) Get(key token.ID) (interface{}, bool) {
	cache.rwlock.RLock()
	defer cache.rwlock.RUnlock()

	item, ok := cache.table[key]
	if !ok {
		return nil, false
	}

	// referenced bit is set to true to indicate that this item is recently accessed.
	atomic.StoreInt32(&item.referenced, 1)

	return item.value, true
}

func (cache *secondChanceCache) Add(key token.ID, value interface{}) {
	cache.rwlock.Lock()
	defer cache.rwlock.Unlock()

	if old, ok := cache.table[key]; ok {
		old.value = value
		atomic.StoreInt32(&old.referenced, 1)
		return
	}

	var item cacheItem
	item.key = key
	item.value = value

	size := len(cache.items)
	num := len(cache.table)
	if num < size {
		// cache is not full, so just store the new item at the end of the list
		cache.table[key] = &item
		cache.items[num] = &item
		return
	}

	// starts victim scan since cache is full
	for {
		// checks whether this item is recently accessed or not
		victim := cache.items[cache.position]
		if atomic.LoadInt32(&victim.referenced) == 0 {
			// a victim is found. delete it, and store the new item here.
			delete(cache.table, victim.key)
			cache.table[key] = &item
			cache.items[cache.position] = &item
			cache.position = (cache.position + 1) % size
			return
		}

		// referenced bit is set to false so that this item will be Get purged
		// unless it is accessed until a next victim scan
		atomic.StoreInt32(&victim.referenced, 0)
		cache.position = (cache.position + 1) % size
	}
}

func (cache *secondChanceCache) Delete(key token.ID) {
	cache.rwlock.Lock()
	defer cache.rwlock.Unlock()

	if old, ok := cache.table[key]; ok {
		old.value = nil
		atomic.StoreInt32(&old.referenced, 1)
		return
	}
}

func (cache *secondChanceCache) clean() {
	cache.rwlock.Lock()
	defer cache.rwlock.Unlock()
	cache.position = 0
	cache.items = make([]*cacheItem, cap(cache.items))
	cache.table = make(map[token.ID]*cacheItem)
}
