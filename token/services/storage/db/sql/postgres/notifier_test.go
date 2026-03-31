/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package postgres

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	herrors "github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver"
	"github.com/jackc/pgxlisten"
	"github.com/stretchr/testify/require"
)

// mockListener implements the databaseListener interface for testing
type mockListener struct {
	ListenFN func(context.Context) error
	HandleFN func(string, pgxlisten.Handler)
}

// Listen calls the wrapped function
func (m *mockListener) Listen(ctx context.Context) error {
	if m.ListenFN != nil {
		return m.ListenFN(ctx)
	}

	return errors.New("mock listen not implemented")
}

// Handle calls the wrapped function
func (m *mockListener) Handle(table string, handler pgxlisten.Handler) {
	if m.HandleFN != nil {
		m.HandleFN(table, handler)
	}
}

// TestNotifierSubscribeError tests that Subscribe returns an error when listener fails to start
func TestNotifierSubscribeError(t *testing.T) {
	// Create a notifier with a listener that will fail during Listen
	listenerErrChan := make(chan error, 1)
	mockListener := &mockListener{
		ListenFN: func(ctx context.Context) error {
			return errors.New("listener failed to start")
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	db := &Notifier{
		table:       "test_table",
		writeDB:     nil, // Not used in this test
		listener:    mockListener,
		listenerErr: listenerErrChan,
		closed:      false,
		ctx:         ctx,
		cancel:      cancel,
	}

	// Subscribe should return the listener error
	err := db.Subscribe(func(operation driver.Operation, m map[driver.ColumnKey]string) {})
	require.Error(t, err)
	require.Contains(t, err.Error(), "listener failed to start")
}

// TestNotifierSubscribeClosed tests that Subscribe returns an error when notifier is closed
func TestNotifierSubscribeClosed(t *testing.T) {
	db := &Notifier{
		table:    "test_table",
		writeDB:  nil,
		listener: &mockListener{},
		closed:   true, // Mark as closed
	}

	// Subscribe should return "notifier is closed" error
	err := db.Subscribe(func(operation driver.Operation, m map[driver.ColumnKey]string) {})
	require.Error(t, err)
	require.Equal(t, herrors.Errorf("notifier is closed").Error(), err.Error())
}

// TestNotifierClose tests that Close properly cleans up resources
func TestNotifierClose(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background()) //nolint:gosec
	db := &Notifier{
		table:            "test_table",
		writeDB:          nil,
		notifyOperations: []driver.Operation{driver.Insert},
		primaryKeys:      []PrimaryKey{},
		listener:         &mockListener{},
		listenerErr:      make(chan error, 1),
		subscribers:      []driver.TriggerCallback{},
		closed:           false,
		ctx:              ctx,
		cancel:           cancel,
	}

	// Add a dummy subscriber to test cleanup
	db.subscribers = append(db.subscribers, func(operation driver.Operation, m map[driver.ColumnKey]string) {})

	// Close should not panic and should set closed flag
	err := db.Close()
	require.NoError(t, err)
	require.True(t, db.closed)
	require.Nil(t, db.subscribers) // Should be nil after Close
}

// TestNotifierListenerErrorChannel tests that ListenerError returns a channel that receives errors
func TestNotifierListenerErrorChannel(t *testing.T) {
	errChan := make(chan error, 1)
	db := &Notifier{
		table:       "test_table",
		writeDB:     nil,
		listener:    &mockListener{},
		listenerErr: errChan,
		closed:      false,
	}

	// Send an error to the channel
	testErr := errors.New("test listener error")
	errChan <- testErr

	// ListenerError should return a channel that receives the error
	listenerErrChan := db.ListenerError()
	select {
	case receivedErr := <-listenerErrChan:
		require.Equal(t, testErr, receivedErr)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timed out waiting for error from ListenerError channel")
	}
}

// TestNotifierSubscribeAfterClose tests that Subscribe returns error when called after Close
func TestNotifierSubscribeAfterClose(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background()) //nolint:gosec
	db := &Notifier{
		table:            "test_table",
		writeDB:          nil,
		notifyOperations: []driver.Operation{driver.Insert},
		primaryKeys:      []PrimaryKey{},
		listener:         &mockListener{},
		listenerErr:      make(chan error, 1),
		subscribers:      []driver.TriggerCallback{},
		closed:           false,
		ctx:              ctx,
		cancel:           cancel,
	}

	// Close the notifier
	err := db.Close()
	require.NoError(t, err)
	require.True(t, db.closed)

	// Subscribe after close should return error
	err = db.Subscribe(func(operation driver.Operation, m map[driver.ColumnKey]string) {})
	require.Error(t, err)
	require.Equal(t, herrors.Errorf("notifier is closed").Error(), err.Error())
}

// TestNotifierSubscriberAccessSafety tests that subscriber access is thread-safe
func TestNotifierSubscriberAccessSafety(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	db := &Notifier{
		table:       "test_table",
		writeDB:     nil,
		listener:    &mockListener{},
		listenerErr: make(chan error, 1),
		closed:      false,
		subscribers: []driver.TriggerCallback{},
		mu:          sync.RWMutex{},
		ctx:         ctx,
		cancel:      cancel,
	}

	// Track callbacks received
	var callbackCount int
	var mutex sync.Mutex

	// Subscribe multiple goroutines
	var wg sync.WaitGroup
	numGoroutines := 10
	subscriptionsPerGoroutine := 100

	for range numGoroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range subscriptionsPerGoroutine {
				_ = db.Subscribe(func(operation driver.Operation, m map[driver.ColumnKey]string) {
					mutex.Lock()
					callbackCount++
					mutex.Unlock()
				})
			}
		}()
	}

	// Wait for all subscriptions to complete
	wg.Wait()

	// Trigger a notification to test thread-safe access
	db.dispatch(driver.Insert, map[driver.ColumnKey]string{"test": "value"})

	// Give time for callbacks to execute
	time.Sleep(50 * time.Millisecond)

	// Verify all subscribers were called
	mutex.Lock()
	expectedCount := numGoroutines * subscriptionsPerGoroutine
	require.Equal(t, expectedCount, callbackCount, "Expected %d callbacks, got %d", expectedCount, callbackCount)
	mutex.Unlock()
}

// TestNotifierConcurrentSubscribeAndClose tests concurrent Subscribe and Close operations
func TestNotifierConcurrentSubscribeAndClose(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background()) //nolint:gosec
	db := &Notifier{
		table:   "test_table",
		writeDB: nil,
		listener: &mockListener{
			ListenFN: func(ctx context.Context) error {
				// Simulate a listener that runs for a short time then returns nil
				ticker := time.NewTicker(20 * time.Millisecond)
				defer ticker.Stop()
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-ticker.C:
					return nil
				}
			},
		},
		listenerErr: make(chan error, 1),
		closed:      false,
		subscribers: []driver.TriggerCallback{},
		mu:          sync.RWMutex{},
		startOnce:   sync.Once{},
		closeOnce:   sync.Once{},
		ctx:         ctx,
		cancel:      cancel,
	}

	var wg sync.WaitGroup
	numSubscribeGoroutines := 5

	// Start goroutines that repeatedly subscribe
	for range numSubscribeGoroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 10 {
				_ = db.Subscribe(func(operation driver.Operation, m map[driver.ColumnKey]string) {})
				time.Sleep(1 * time.Millisecond)
			}
		}()
	}

	// Start a goroutine that closes the notifier after a short delay
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(30 * time.Millisecond) // Wait for some subscriptions to happen
		_ = db.Close()
	}()

	// Wait for all goroutines to complete
	wg.Wait()

	// After close, subscribers should be nil
	require.Nil(t, db.subscribers)
	require.True(t, db.closed)
}

func TestNotifierPayloadParsing(t *testing.T) {
	h := &notificationHandler{
		table: "test_table",
		primaryKeys: []PrimaryKey{
			{name: "id1", valueDecoder: identity},
			{name: "id2", valueDecoder: identity},
		},
	}

	// Test valid INSERT
	payload := `["INSERT", "val1", "val2"]`
	op, m, err := h.parsePayload(payload)
	require.NoError(t, err)
	require.Equal(t, driver.Insert, op)
	require.Equal(t, map[driver.ColumnKey]string{"id1": "val1", "id2": "val2"}, m)

	// Test valid UPDATE
	payload = `["UPDATE", "val1", "val2"]`
	op, m, err = h.parsePayload(payload)
	require.NoError(t, err)
	require.Equal(t, driver.Update, op)
	require.Equal(t, map[driver.ColumnKey]string{"id1": "val1", "id2": "val2"}, m)

	// Test valid DELETE
	payload = `["DELETE", "val1", "val2"]`
	op, m, err = h.parsePayload(payload)
	require.NoError(t, err)
	require.Equal(t, driver.Delete, op)
	require.Equal(t, map[driver.ColumnKey]string{"id1": "val1", "id2": "val2"}, m)

	// Test malformed JSON
	payload = `["INSERT", "val1"`
	_, _, err = h.parsePayload(payload)
	require.Error(t, err)

	// Test wrong number of items
	payload = `["INSERT", "val1"]`
	_, _, err = h.parsePayload(payload)
	require.Error(t, err)

	// Test unknown operation
	payload = `["UNKNOWN", "val1", "val2"]`
	_, _, err = h.parsePayload(payload)
	require.Error(t, err)
}

func TestNotifierGetSchema(t *testing.T) {
	db := &Notifier{
		table:            "test_table",
		notifyOperations: []driver.Operation{driver.Insert},
		primaryKeys: []PrimaryKey{
			{name: "id1", valueDecoder: identity},
		},
	}

	schema := db.GetSchema()
	require.Contains(t, schema, `output = json_build_array(TG_OP, row."id1"::text)::text;`)
	require.Contains(t, schema, `CREATE OR REPLACE TRIGGER "trigger_test_table"`)
	require.Contains(t, schema, `AFTER INSERT ON test_table`)

	// Test with schema-qualified table name
	db.table = "public.test_table"
	schema = db.GetSchema()
	require.Contains(t, schema, `CREATE OR REPLACE TRIGGER "trigger_public.test_table"`)
	require.Contains(t, schema, `AFTER INSERT ON public.test_table`)
}
