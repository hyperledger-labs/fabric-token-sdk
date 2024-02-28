/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import "sync"

type LazyGetter[V any] struct {
	v        V
	provider func() V
	once     sync.Once
}

func NewLazyGetter[V any](provider func() V) *LazyGetter[V] {
	return &LazyGetter[V]{provider: provider}
}

func (g *LazyGetter[V]) Get() V {
	g.once.Do(func() {
		g.v = g.provider()
	})
	return g.v
}
