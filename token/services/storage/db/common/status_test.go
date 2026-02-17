/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
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

	if ok {
		t.Fatalf("Map entry should be deleted when no listeners remain, but found %d listeners", len(listeners))
	}
}

func TestNotify(t *testing.T) {
	ss := NewStatusSupport()
	txID := "tx456"
	ch := make(chan StatusEvent, 1)
	ss.AddStatusListener(txID, ch)

	event := StatusEvent{
		Ctx:               t.Context(),
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
	_, ok := ss.listeners[txID]
	if ok {
		t.Fatalf("Map entry should be deleted after removing last listener to free memory")
	}
}

func TestAddAndDeleteMultipleStatusListeners(t *testing.T) {
	ss := NewStatusSupport()

	txID := "txMulti"
	var channels []chan StatusEvent

	// Add multiple listeners
	for i := range 5 {
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

	// After deletion, the map entry should be removed
	ss.mutex.RLock()
	_, ok = ss.listeners[txID]
	ss.mutex.RUnlock()

	if ok {
		t.Fatalf("Map entry should be deleted after removing all listeners")
	}
}

func TestNotifyMultipleListeners(t *testing.T) {
	ss := NewStatusSupport()
	txID := "txNotifyMulti"
	var channels []chan StatusEvent

	// Add multiple listeners
	for range 3 {
		ch := make(chan StatusEvent, 1)
		channels = append(channels, ch)
		ss.AddStatusListener(txID, ch)
	}

	event := StatusEvent{
		Ctx:               t.Context(),
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

func TestMapCleanupAfterManyTransactions(t *testing.T) {
	// Verify that map entries are properly cleaned up after many add/delete cycles
	ss := NewStatusSupport()

	// Simulate many transactions adding and removing listeners
	numTransactions := 1000

	for i := range numTransactions {
		txID := "tx" + string(rune(i))
		ch := make(chan StatusEvent, 1)

		ss.AddStatusListener(txID, ch)
		ss.DeleteStatusListener(txID, ch)
	}

	// Map should be empty after all listeners are removed
	ss.mutex.RLock()
	mapSize := len(ss.listeners)
	ss.mutex.RUnlock()

	if mapSize != 0 {
		t.Fatalf("Map not properly cleaned up: has %d entries after cleanup, expected 0", mapSize)
	}

	t.Logf("Map properly cleaned up: %d entries (expected 0)", mapSize)
}

func TestMapGrowthWithMultipleListenersPerTx(t *testing.T) {
	// Test that map entries are only deleted when ALL listeners are removed
	ss := NewStatusSupport()
	txID := "txShared"

	ch1 := make(chan StatusEvent, 1)
	ch2 := make(chan StatusEvent, 1)
	ch3 := make(chan StatusEvent, 1)

	// Add 3 listeners for same transaction
	ss.AddStatusListener(txID, ch1)
	ss.AddStatusListener(txID, ch2)
	ss.AddStatusListener(txID, ch3)

	ss.mutex.RLock()
	listeners := ss.listeners[txID]
	ss.mutex.RUnlock()

	if len(listeners) != 3 {
		t.Fatalf("Expected 3 listeners, got %d", len(listeners))
	}

	// Remove first listener - map entry should still exist
	ss.DeleteStatusListener(txID, ch1)

	ss.mutex.RLock()
	_, ok := ss.listeners[txID]
	listenerCount := len(ss.listeners[txID])
	ss.mutex.RUnlock()

	if !ok {
		t.Fatalf("Map entry should exist after removing 1 of 3 listeners")
	}
	if listenerCount != 2 {
		t.Fatalf("Expected 2 listeners remaining, got %d", listenerCount)
	}

	// Remove second listener - map entry should still exist
	ss.DeleteStatusListener(txID, ch2)

	ss.mutex.RLock()
	_, ok = ss.listeners[txID]
	listenerCount = len(ss.listeners[txID])
	ss.mutex.RUnlock()

	if !ok {
		t.Fatalf("Map entry should exist after removing 2 of 3 listeners")
	}
	if listenerCount != 1 {
		t.Fatalf("Expected 1 listener remaining, got %d", listenerCount)
	}

	// Remove last listener - NOW map entry should be deleted
	ss.DeleteStatusListener(txID, ch3)

	ss.mutex.RLock()
	_, ok = ss.listeners[txID]
	ss.mutex.RUnlock()

	if ok {
		t.Fatalf("Map entry should be deleted after removing last listener")
	}

	t.Logf("Map entry correctly maintained until last listener removed")
}
