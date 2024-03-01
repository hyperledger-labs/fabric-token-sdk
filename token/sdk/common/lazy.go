/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import "sync"

type LazyGetter[V any] struct {
	v        V
	err      error
	provider func() (V, error)
	once     sync.Once
}

func NewLazyGetter[V any](provider func() (V, error)) *LazyGetter[V] {
	return &LazyGetter[V]{provider: provider}
}

func (g *LazyGetter[V]) Get() (V, error) {
	g.once.Do(func() {
		g.v, g.err = g.provider()
	})
	return g.v, g.err
}
