/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"context"
	"sync"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
)

// StatusEvent models an event related to the status of a transaction
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
			// Zero out the reference before slicing
			// to allow the garbage collector to reclaim the memory
			ls[i] = nil
			ls = append(ls[:i], ls[i+1:]...)
			c.listeners[txID] = ls
			return
		}
	}
}

func (c *StatusSupport) Notify(event StatusEvent) {
	logger.DebugfContext(event.Ctx, "Start notify for [%s]", event.TxID)
	defer logger.DebugfContext(event.Ctx, "Notified for [%s]", event.TxID)
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
