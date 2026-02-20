/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package lookup_test

import (
	"context"
	"testing"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabricx/lookup"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabricx/lookup/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNSListenerManager tests the registration of lookup listeners.
func TestNSListenerManager(t *testing.T) {
	q := &mock.Queue{}
	qs := &mock.QueryService{}
	mgr := lookup.NewNSListenerManager(q, qs)

	assert.True(t, mgr.PermanentLookupListenerSupported())

	l1 := &mock.Listener{}
	// Test adding a permanent lookup listener
	err := mgr.AddPermanentLookupListener("ns", "key", l1)
	require.NoError(t, err)
	assert.Equal(t, 1, q.EnqueueCallCount())
	assert.IsType(t, &lookup.PermanentKeyCheck{}, q.EnqueueArgsForCall(0))

	l2 := &mock.Listener{}
	// Test adding a one-time lookup listener
	err = mgr.AddLookupListener("ns", "key", l2)
	require.NoError(t, err)
	assert.Equal(t, 2, q.EnqueueCallCount())
	assert.IsType(t, &lookup.KeyCheck{}, q.EnqueueArgsForCall(1))

	// Test removing a lookup listener
	err = mgr.RemoveLookupListener("key", l1)
	require.NoError(t, err)

	err = mgr.RemoveLookupListener("key", l2)
	require.NoError(t, err)

	// Removing non-existent
	err = mgr.RemoveLookupListener("key", l1)
	require.NoError(t, err)
}

// TestKeyCheck_Process tests the execution of a one-time key check event.
func TestKeyCheck_Process(t *testing.T) {
	ctx := context.Background()

	t.Run("Found", func(t *testing.T) {
		// Key is found immediately
		q := &mock.Queue{}
		qs := &mock.QueryService{}
		qs.GetStateReturns(&driver2.VaultValue{Raw: []byte("value")}, nil)
		mgr := lookup.NewNSListenerManager(q, qs)
		l := &mock.Listener{}
		require.NoError(t, mgr.AddLookupListener("ns", "key", l))

		kc := q.EnqueueArgsForCall(0).(*lookup.KeyCheck)

		err := kc.Process(ctx)
		require.NoError(t, err)
		assert.Equal(t, 1, l.OnStatusCallCount())
		assert.Equal(t, 1, q.EnqueueCallCount()) // Initial enqueue only
	})

	t.Run("Removed", func(t *testing.T) {
		q := &mock.Queue{}
		qs := &mock.QueryService{}
		mgr := lookup.NewNSListenerManager(q, qs)
		l := &mock.Listener{}
		require.NoError(t, mgr.AddLookupListener("ns", "key", l))

		kc := q.EnqueueArgsForCall(0).(*lookup.KeyCheck)

		// Remove before processing
		require.NoError(t, mgr.RemoveLookupListener("key", l))

		err := kc.Process(ctx)
		require.NoError(t, err)
		assert.Equal(t, 0, l.OnStatusCallCount())
		assert.Equal(t, 0, qs.GetStateCallCount())
	})

	t.Run("NotFound_Reschedule", func(t *testing.T) {
		// Key not found, should reschedule
		q := &mock.Queue{}
		qs := &mock.QueryService{}
		qs.GetStateReturns(nil, nil)
		mgr := lookup.NewNSListenerManager(q, qs)
		l := &mock.Listener{}
		require.NoError(t, mgr.AddLookupListener("ns", "key", l))

		kc := q.EnqueueArgsForCall(0).(*lookup.KeyCheck)
		kc.Interval = time.Millisecond

		err := kc.Process(ctx)
		require.NoError(t, err)
		assert.Equal(t, 0, l.OnStatusCallCount())

		// Wait for the async reschedule to happen
		assert.Eventually(t, func() bool {
			return q.EnqueueCallCount() == 2 // 1 from Add, 1 from Reschedule
		}, 200*time.Millisecond, 10*time.Millisecond)
	})

	t.Run("NotFound_DeadlineReached", func(t *testing.T) {
		// Key not found and deadline is reached, should notify error
		q := &mock.Queue{}
		qs := &mock.QueryService{}
		qs.GetStateReturns(nil, nil)
		mgr := lookup.NewNSListenerManager(q, qs)
		l := &mock.Listener{}
		require.NoError(t, mgr.AddLookupListener("ns", "key", l))

		kc := q.EnqueueArgsForCall(0).(*lookup.KeyCheck)
		kc.Deadline = time.Now().Add(-time.Minute)

		err := kc.Process(ctx)
		require.NoError(t, err)
		assert.Equal(t, 0, l.OnStatusCallCount())
		assert.Equal(t, 1, l.OnErrorCallCount())
		assert.Equal(t, 1, q.EnqueueCallCount()) // only the initial one
	})
}

// TestPermanentKeyCheck_Process tests the execution of a permanent key check event.
func TestPermanentKeyCheck_Process(t *testing.T) {
	ctx := context.Background()

	t.Run("NotifyOnChange", func(t *testing.T) {
		q := &mock.Queue{}
		qs := &mock.QueryService{}
		// 1st call: new value
		qs.GetStateReturnsOnCall(0, &driver2.VaultValue{Raw: []byte("v1")}, nil)
		// 2nd call: same value
		qs.GetStateReturnsOnCall(1, &driver2.VaultValue{Raw: []byte("v1")}, nil)
		// 3rd call: changed value
		qs.GetStateReturnsOnCall(2, &driver2.VaultValue{Raw: []byte("v2")}, nil)

		mgr := lookup.NewNSListenerManager(q, qs)
		l := &mock.Listener{}
		require.NoError(t, mgr.AddPermanentLookupListener("ns", "key", l))

		pkc := q.EnqueueArgsForCall(0).(*lookup.PermanentKeyCheck)
		pkc.Interval = time.Millisecond

		// First call - should notify status
		err := pkc.Process(ctx)
		require.NoError(t, err)
		assert.Equal(t, 1, l.OnStatusCallCount())

		// Second call - same value, should NOT notify status again
		err = pkc.Process(ctx)
		require.NoError(t, err)
		assert.Equal(t, 1, l.OnStatusCallCount())

		// Third call - value changed, should notify status again
		err = pkc.Process(ctx)
		require.NoError(t, err)
		assert.Equal(t, 2, l.OnStatusCallCount())

		// Should always reschedule regardless of value change
		assert.Eventually(t, func() bool {
			return q.EnqueueCallCount() >= 2
		}, 200*time.Millisecond, 10*time.Millisecond)
	})

	t.Run("Removed", func(t *testing.T) {
		q := &mock.Queue{}
		qs := &mock.QueryService{}
		mgr := lookup.NewNSListenerManager(q, qs)
		l := &mock.Listener{}
		require.NoError(t, mgr.AddPermanentLookupListener("ns", "key", l))

		pkc := q.EnqueueArgsForCall(0).(*lookup.PermanentKeyCheck)

		// Remove
		require.NoError(t, mgr.RemoveLookupListener("key", l))

		err := pkc.Process(ctx)
		require.NoError(t, err)
		assert.Equal(t, 0, l.OnStatusCallCount())
		assert.Equal(t, 0, qs.GetStateCallCount())
		assert.Equal(t, 1, q.EnqueueCallCount()) // only the initial one
	})
}

// TestOnlyOnceListener ensures that a listener is notified only once.
func TestOnlyOnceListener(t *testing.T) {
	l := &mock.Listener{}
	o := lookup.NewOnlyOnceListener(l)
	ctx := context.Background()

	// Notify status twice
	o.OnStatus(ctx, "key", []byte("v"))
	o.OnStatus(ctx, "key", []byte("v"))
	// Notify error once
	o.OnError(ctx, "key", errors.New("err"))

	// Only the first notification should go through
	assert.Equal(t, 1, l.OnStatusCallCount())
	assert.Equal(t, 0, l.OnErrorCallCount())
}

// TestNSListenerManagerProvider tests the creation of a listener manager.
func TestNSListenerManagerProvider(t *testing.T) {
	q := &mock.Queue{}
	qs := &mock.QueryService{}
	qsp := &mock.QueryServiceProvider{}
	qsp.GetReturns(qs, nil)

	p := lookup.NewQueryServiceBased(qsp, q)

	// Successful creation
	mgr, err := p.NewManager("network", "channel")
	require.NoError(t, err)
	assert.NotNil(t, mgr)

	// Error scenario: query service provider fails
	qsp.GetReturns(nil, errors.New("error"))
	mgr, err = p.NewManager("network", "channel")
	require.Error(t, err)
	assert.Nil(t, mgr)
}
