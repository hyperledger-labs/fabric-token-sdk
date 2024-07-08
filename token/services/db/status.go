/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package db

import (
	"context"
	"sync"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"go.opentelemetry.io/otel/trace"
)

type StatusEvent struct {
	Ctx               context.Context
	TxID              string
	ValidationCode    driver.TxStatus
	ValidationMessage string
}

type StatusSupport struct {
	listeners      map[string][]chan StatusEvent
	mutex          sync.RWMutex
	pollingTimeout time.Duration
}

func NewStatusSupport() *StatusSupport {
	return &StatusSupport{
		listeners:      map[string][]chan StatusEvent{},
		pollingTimeout: 1 * time.Second,
	}
}

func (c *StatusSupport) AddStatusListener(txID string, ch chan StatusEvent) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	ls, ok := c.listeners[txID]
	if !ok {
		ls = []chan StatusEvent{}
	}
	ls = append(ls, ch)
	c.listeners[txID] = ls
}

func (c *StatusSupport) DeleteStatusListener(txID string, ch chan StatusEvent) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	ls, ok := c.listeners[txID]
	if !ok {
		return
	}
	for i, l := range ls {
		if l == ch {
			ls = append(ls[:i], ls[i+1:]...)
			c.listeners[txID] = ls
			return
		}
	}
}

func (c *StatusSupport) Notify(event StatusEvent) {
	span := trace.SpanFromContext(event.Ctx)
	span.AddEvent("start_notify")
	defer span.AddEvent("end_notify")
	c.mutex.RLock()
	listeners := c.listeners[event.TxID]
	if len(listeners) == 0 {
		c.mutex.RUnlock()
		return
	}
	// clone listeners and release lock
	clone := make([]chan StatusEvent, len(listeners))
	copy(clone, listeners)
	c.mutex.RUnlock()

	for _, listener := range clone {
		listener <- event
	}
}
