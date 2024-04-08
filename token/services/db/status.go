/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package db

import (
	"sync"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
)

type StatusEvent struct {
	TxID              string
	ValidationCode    driver.TxStatus
	ValidationMessage string
}

type StatusSupport struct {
	listeners      map[string][]chan StatusEvent
	mutex          sync.Mutex
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
		c.listeners[txID] = ls
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
	c.mutex.Lock()
	defer c.mutex.Unlock()

	listeners := c.listeners[event.TxID]
	for _, listener := range listeners {
		listener <- event
	}
}
