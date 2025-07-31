/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cache

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils"
)

// NoCache implements a dummy cache that does nothing
type NoCache[T any] struct {
}

// NewNoCache returns a new instance of NoCache
func NewNoCache[T any]() *NoCache[T] {
	return &NoCache[T]{}
}

func (n *NoCache[T]) Get(key string) (T, bool) {
	return utils.Zero[T](), false
}

func (n *NoCache[T]) GetOrLoad(key string, loader func() (T, error)) (T, bool, error) {
	v, err := loader()
	return v, false, err
}

func (n *NoCache[T]) Add(key string, value T) {
}

func (n *NoCache[T]) Delete(key string) {
}
