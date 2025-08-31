/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"context"
	"testing"
	"time"
)

func TestAddAndDeleteStatusListener(t *testing.T) {
	ss := NewStatusSupport()
	txID := "tx123"
	ch := make(chan StatusEvent, 1)

	// Add listener
	ss.AddStatusListener(txID, ch)

	ss.mutex.RLock()
	listeners, ok := ss.listeners[txID]
	ss.mutex.RUnlock()

	if !ok || len(listeners) != 1 || listeners[0] != ch {
		t.Fatalf("Listener was not added correctly")
	}

	// Delete listener
	ss.DeleteStatusListener(txID, ch)

	ss.mutex.RLock()
	listeners, ok = ss.listeners[txID]
	ss.mutex.RUnlock()

	if !ok || len(listeners) != 0 {
		t.Fatalf("Listener was not deleted correctly")
	}
}

func TestNotify(t *testing.T) {
	ss := NewStatusSupport()
	txID := "tx456"
	ch := make(chan StatusEvent, 1)
	ss.AddStatusListener(txID, ch)

	event := StatusEvent{
		Ctx:               context.Background(),
		TxID:              txID,
		ValidationCode:    0,
		ValidationMessage: "ok",
	}

	// Notify listeners in a goroutine since Notify sends on channel
	go ss.Notify(event)

	select {
	case e := <-ch:
		if e.TxID != txID {
			t.Fatalf("Unexpected TxID in event: %s", e.TxID)
		}
	case <-time.After(1 * time.Second):
		t.Fatalf("Timeout waiting for event notification")
	}

	ss.DeleteStatusListener(txID, ch)
}

func TestMemoryReleaseAfterDelete(t *testing.T) {
	ss := NewStatusSupport()
	txID := "tx789"
	ch := make(chan StatusEvent, 1)

	ss.AddStatusListener(txID, ch)
	ss.DeleteStatusListener(txID, ch)

	ss.mutex.RLock()
	defer ss.mutex.RUnlock()
	listeners, ok := ss.listeners[txID]
	if ok && len(listeners) != 0 {
		t.Fatalf("Listeners should be empty after delete to free memory")
	}
}

func TestAddAndDeleteMultipleStatusListeners(t *testing.T) {
	ss := NewStatusSupport()

	txID := "txMulti"
	var channels []chan StatusEvent

	// Add multiple listeners
	for i := 0; i < 5; i++ {
		ch := make(chan StatusEvent, i+1)
		channels = append(channels, ch)
		ss.AddStatusListener(txID, ch)
	}

	ss.mutex.RLock()
	listeners, ok := ss.listeners[txID]
	ss.mutex.RUnlock()

	if !ok || len(listeners) != 5 {
		t.Fatalf("Expected 5 listeners, got %d", len(listeners))
	}

	// Remove each listener one by one
	for _, ch := range channels {
		ss.DeleteStatusListener(txID, ch)
	}

	// After deletion, the list should be empty
	ss.mutex.RLock()
	listeners, ok = ss.listeners[txID]
	ss.mutex.RUnlock()

	if !ok || len(listeners) != 0 {
		t.Fatalf("Listeners not properly deleted, expected 0 got %d", len(listeners))
	}
}

func TestNotifyMultipleListeners(t *testing.T) {
	ss := NewStatusSupport()
	txID := "txNotifyMulti"
	var channels []chan StatusEvent

	// Add multiple listeners
	for i := 0; i < 3; i++ {
		ch := make(chan StatusEvent, 1)
		channels = append(channels, ch)
		ss.AddStatusListener(txID, ch)
	}

	event := StatusEvent{
		Ctx:               context.Background(),
		TxID:              txID,
		ValidationCode:    0,
		ValidationMessage: "multi notify",
	}

	go ss.Notify(event)

	// Verify that all listeners receive the event
	for _, ch := range channels {
		select {
		case e := <-ch:
			if e.TxID != txID {
				t.Fatalf("Unexpected TxID in event: %s", e.TxID)
			}
		case <-time.After(1 * time.Second):
			t.Fatalf("Timeout waiting for event notification on one of the channels")
		}
	}

	// Cleanup
	for _, ch := range channels {
		ss.DeleteStatusListener(txID, ch)
	}
}
