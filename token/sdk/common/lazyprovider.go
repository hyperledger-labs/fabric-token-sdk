/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils"
)

type LazyProvider[I any, V any] interface {
	Get(I) (V, error)
	Update(I) (V, V, error)
	Delete(I) (V, bool)
	Length() int
}

func NewLazyProvider[K comparable, V any](provider func(K) (V, error)) *lazyProvider[K, K, V] {
	return NewLazyProviderWithKeyMapper[K, K, V](func(k K) K { return k }, provider)
}

func NewLazyProviderWithKeyMapper[I any, K comparable, V any](keyMapper func(I) K, provider func(I) (V, error)) *lazyProvider[I, K, V] {
	return &lazyProvider[I, K, V]{
		cache:     make(map[K]V),
		provider:  provider,
		keyMapper: keyMapper,
		zero:      utils.Zero[V](),
	}
}

type lazyProvider[I any, K comparable, V any] struct {
	cache     map[K]V
	cacheLock sync.RWMutex
	keyMapper func(I) K
	provider  func(I) (V, error)
	zero      V
}

func (v *lazyProvider[I, K, V]) Update(input I) (V, V, error) {
	key := v.keyMapper(input)

	v.cacheLock.Lock()
	defer v.cacheLock.Unlock()
	oldRes := v.cache[key]

	// create the service for the new public params
	res, err := v.provider(input)
	if err != nil {
		return v.zero, v.zero, err
	}

	// register the new service
	v.cache[key] = res

	return oldRes, res, nil

}

func (v *lazyProvider[I, K, V]) Get(input I) (V, error) {
	key := v.keyMapper(input)
	// Check cache
	v.cacheLock.RLock()
	res, ok := v.cache[key]
	v.cacheLock.RUnlock()
	if ok {
		return res, nil
	}

	// lock
	v.cacheLock.Lock()
	defer v.cacheLock.Unlock()

	// check cache again
	res, ok = v.cache[key]
	if ok {
		return res, nil
	}

	// update cache
	res, err := v.provider(input)
	if err != nil {
		return v.zero, err
	}
	v.cache[key] = res

	return res, nil
}

func (v *lazyProvider[I, K, V]) Delete(input I) (V, bool) {
	key := v.keyMapper(input)

	v.cacheLock.RLock()
	res, ok := v.cache[key]
	v.cacheLock.RUnlock()

	if ok {
		v.cacheLock.Lock()
		delete(v.cache, key)
		v.cacheLock.Unlock()
	}

	return res, ok
}

func (v *lazyProvider[I, K, V]) Length() int {
	return len(v.cache)
}
