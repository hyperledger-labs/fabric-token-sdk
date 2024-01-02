/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mailman

import (
	"github.com/gammazero/deque"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type Queue interface {
	Push(t *token.ID) bool
	Pop() (*token.ID, bool)
	Remove(t *token.ID) bool
}

type queue struct {
	data *deque.Deque[*token.ID]
}

func NewQueue(size int) *queue {
	return &queue{data: deque.New[*token.ID](size, size/2)}
}

func (q *queue) Push(t *token.ID) bool {
	if t == nil {
		return false
	}
	q.data.PushBack(t)
	return true
}

func (q *queue) Pop() (*token.ID, bool) {
	if q.data.Len() == 0 {
		return nil, false
	}
	return q.data.PopFront(), true
}

func (q *queue) Remove(t *token.ID) bool {
	if t == nil {
		return false
	}
	idx := q.data.Index(func(it *token.ID) bool {
		return it == t
	})
	if idx > -1 {
		q.data.Remove(idx)
		return true
	}

	return false
}
