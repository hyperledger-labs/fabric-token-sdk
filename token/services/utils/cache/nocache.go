/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cache

type NoCache[T any] struct {
}

func NewNoCache[T any]() *NoCache[T] {
	return &NoCache[T]{}
}

func (n *NoCache[T]) Get(key string) (T, bool) {
	return Zero[T](), false
}

func (n *NoCache[T]) GetOrLoad(key string, loader func() (T, error)) (T, bool, error) {
	v, err := loader()
	return v, false, err
}

func (n *NoCache[T]) Add(key string, value T) {
	return
}

func (n *NoCache[T]) Delete(key string) {
	return
}

func Zero[T any]() T {
	var result T
	return result
}
