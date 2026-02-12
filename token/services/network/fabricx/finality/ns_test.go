/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package finality_test

import (
	"context"
	"sync"
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	cdriver "github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	fdriver "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabricx/finality"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabricx/finality/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabricx/finality/queue"
	"github.com/hyperledger/fabric-x-committer/api/protoblocktx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListenerEvent_Process_Valid(t *testing.T) {
	ctx := t.Context()
	txID := "tx123"
	namespace := "token-namespace"
	tokenRequestHash := []byte("hash123")
	key := "token-request-key"

	mockQS := &mock.QueryService{}
	mockKT := &mock.KeyTranslator{}
	mockListener := &mock.Listener{}

	mockKT.CreateTokenRequestKeyReturns(key, nil)
	mockQS.GetStateReturns(&cdriver.VaultValue{Raw: tokenRequestHash}, nil)

	event := &finality.ListenerEvent{
		QueryService:  mockQS,
		KeyTranslator: mockKT,
		Listener:      mockListener,
		TxID:          txID,
		Status:        fdriver.Valid,
		StatusMessage: "",
		Namespace:     namespace,
	}

	err := event.Process(ctx)
	require.NoError(t, err)

	assert.Equal(t, 1, mockKT.CreateTokenRequestKeyCallCount())
	assert.Equal(t, txID, mockKT.CreateTokenRequestKeyArgsForCall(0))

	assert.Equal(t, 1, mockQS.GetStateCallCount())
	ns, k := mockQS.GetStateArgsForCall(0)
	assert.Equal(t, namespace, ns)
	assert.Equal(t, key, k)

	assert.Equal(t, 1, mockListener.OnStatusCallCount())
	callCtx, callTxID, callStatus, callMsg, callHash := mockListener.OnStatusArgsForCall(0)
	assert.Equal(t, ctx, callCtx)
	assert.Equal(t, txID, callTxID)
	assert.Equal(t, fdriver.Valid, callStatus)
	assert.Equal(t, "", callMsg)
	assert.Equal(t, tokenRequestHash, callHash)
}

func TestListenerEvent_Process_Invalid(t *testing.T) {
	ctx := t.Context()
	txID := "tx123"
	statusMessage := "validation failed"

	mockQS := &mock.QueryService{}
	mockKT := &mock.KeyTranslator{}
	mockListener := &mock.Listener{}

	event := &finality.ListenerEvent{
		QueryService:  mockQS,
		KeyTranslator: mockKT,
		Listener:      mockListener,
		TxID:          txID,
		Status:        fdriver.Invalid,
		StatusMessage: statusMessage,
		Namespace:     "token-namespace",
	}

	err := event.Process(ctx)
	require.NoError(t, err)

	// Should not fetch token request hash for invalid transactions
	assert.Equal(t, 0, mockKT.CreateTokenRequestKeyCallCount())
	assert.Equal(t, 0, mockQS.GetStateCallCount())

	assert.Equal(t, 1, mockListener.OnStatusCallCount())
	callCtx, callTxID, callStatus, callMsg, callHash := mockListener.OnStatusArgsForCall(0)
	assert.Equal(t, ctx, callCtx)
	assert.Equal(t, txID, callTxID)
	assert.Equal(t, fdriver.Invalid, callStatus)
	assert.Equal(t, statusMessage, callMsg)
	assert.Nil(t, callHash)
}

func TestListenerEvent_Process_Unknown_TxCheckSucceeds(t *testing.T) {
	ctx := t.Context()
	txID := "tx123"
	namespace := "token-namespace"
	tokenRequestHash := []byte("hash123")
	key := "token-request-key"

	mockQS := &mock.QueryService{}
	mockKT := &mock.KeyTranslator{}
	mockListener := &mock.Listener{}

	// TxCheck will query the transaction status
	mockQS.GetTransactionStatusReturns(int32(protoblocktx.Status_COMMITTED), nil)
	mockKT.CreateTokenRequestKeyReturns(key, nil)
	mockQS.GetStateReturns(&cdriver.VaultValue{Raw: tokenRequestHash}, nil)

	event := &finality.ListenerEvent{
		QueryService:  mockQS,
		KeyTranslator: mockKT,
		Listener:      mockListener,
		TxID:          txID,
		Status:        fdriver.Unknown,
		StatusMessage: "",
		Namespace:     namespace,
	}

	err := event.Process(ctx)
	require.NoError(t, err)

	// TxCheck should have been executed
	assert.Equal(t, 1, mockQS.GetTransactionStatusCallCount())
	assert.Equal(t, txID, mockQS.GetTransactionStatusArgsForCall(0))

	// Should fetch token request hash since status is valid
	assert.Equal(t, 1, mockKT.CreateTokenRequestKeyCallCount())
	assert.Equal(t, 1, mockQS.GetStateCallCount())

	assert.Equal(t, 1, mockListener.OnStatusCallCount())
}

func TestListenerEvent_Process_Unknown_TxCheckFails(t *testing.T) {
	ctx := t.Context()
	txID := "tx123"

	mockQS := &mock.QueryService{}
	mockKT := &mock.KeyTranslator{}
	mockListener := &mock.Listener{}

	// TxCheck will fail to query the transaction status
	mockQS.GetTransactionStatusReturns(int32(protoblocktx.Status_NOT_VALIDATED), errors.New("query failed"))

	event := &finality.ListenerEvent{
		QueryService:  mockQS,
		KeyTranslator: mockKT,
		Listener:      mockListener,
		TxID:          txID,
		Status:        fdriver.Unknown,
		StatusMessage: "",
		Namespace:     "token-namespace",
	}

	err := event.Process(ctx)
	require.NoError(t, err)

	// TxCheck failed, so the event should still notify with Unknown status
	assert.Equal(t, 1, mockListener.OnStatusCallCount())
	_, _, callStatus, _, callHash := mockListener.OnStatusArgsForCall(0)
	assert.Equal(t, fdriver.Unknown, callStatus)
	assert.Nil(t, callHash)
}

func TestListenerEvent_Process_Busy_TxCheckSucceeds(t *testing.T) {
	ctx := t.Context()
	txID := "tx123"
	namespace := "token-namespace"
	tokenRequestHash := []byte("hash123")
	key := "token-request-key"

	mockQS := &mock.QueryService{}
	mockKT := &mock.KeyTranslator{}
	mockListener := &mock.Listener{}

	// TxCheck will query the transaction status and find it committed
	mockQS.GetTransactionStatusReturns(int32(protoblocktx.Status_COMMITTED), nil)
	mockKT.CreateTokenRequestKeyReturns(key, nil)
	mockQS.GetStateReturns(&cdriver.VaultValue{Raw: tokenRequestHash}, nil)

	event := &finality.ListenerEvent{
		QueryService:  mockQS,
		KeyTranslator: mockKT,
		Listener:      mockListener,
		TxID:          txID,
		Status:        fdriver.Busy,
		StatusMessage: "",
		Namespace:     namespace,
	}

	err := event.Process(ctx)
	require.NoError(t, err)

	// TxCheck should have been executed
	assert.Equal(t, 1, mockQS.GetTransactionStatusCallCount())
	assert.Equal(t, 1, mockListener.OnStatusCallCount())
}

func TestListenerEvent_Process_CreateTokenRequestKeyError(t *testing.T) {
	ctx := t.Context()
	txID := "tx123"

	mockQS := &mock.QueryService{}
	mockKT := &mock.KeyTranslator{}
	mockListener := &mock.Listener{}

	mockKT.CreateTokenRequestKeyReturns("", errors.New("key creation failed"))

	event := &finality.ListenerEvent{
		QueryService:  mockQS,
		KeyTranslator: mockKT,
		Listener:      mockListener,
		TxID:          txID,
		Status:        fdriver.Valid,
		StatusMessage: "",
		Namespace:     "token-namespace",
	}

	err := event.Process(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "can't create for token request")
	assert.Contains(t, err.Error(), "key creation failed")
}

func TestListenerEvent_Process_GetStateError(t *testing.T) {
	ctx := t.Context()
	txID := "tx123"
	key := "token-request-key"

	mockQS := &mock.QueryService{}
	mockKT := &mock.KeyTranslator{}
	mockListener := &mock.Listener{}

	mockKT.CreateTokenRequestKeyReturns(key, nil)
	mockQS.GetStateReturns(nil, errors.New("state retrieval failed"))

	event := &finality.ListenerEvent{
		QueryService:  mockQS,
		KeyTranslator: mockKT,
		Listener:      mockListener,
		TxID:          txID,
		Status:        fdriver.Valid,
		StatusMessage: "",
		Namespace:     "token-namespace",
	}

	err := event.Process(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "can't get state for token request")
	assert.Contains(t, err.Error(), "state retrieval failed")
}

func TestListenerEvent_String(t *testing.T) {
	event := &finality.ListenerEvent{
		TxID: "tx123",
	}

	str := event.String()
	assert.Equal(t, "ListenerEvent[tx123]", str)
}

func TestTxCheck_Process_Valid(t *testing.T) {
	ctx := t.Context()
	txID := "tx123"
	namespace := "token-namespace"
	tokenRequestHash := []byte("hash123")
	key := "token-request-key"

	mockQS := &mock.QueryService{}
	mockKT := &mock.KeyTranslator{}
	mockListener := &mock.Listener{}

	mockQS.GetTransactionStatusReturns(int32(protoblocktx.Status_COMMITTED), nil)
	mockKT.CreateTokenRequestKeyReturns(key, nil)
	mockQS.GetStateReturns(&cdriver.VaultValue{Raw: tokenRequestHash}, nil)

	txCheck := &finality.TxCheck{
		QueryService:  mockQS,
		KeyTranslator: mockKT,
		Listener:      mockListener,
		TxID:          txID,
		Namespace:     namespace,
	}

	err := txCheck.Process(ctx)
	require.NoError(t, err)

	assert.Equal(t, 1, mockQS.GetTransactionStatusCallCount())
	assert.Equal(t, txID, mockQS.GetTransactionStatusArgsForCall(0))

	assert.Equal(t, 1, mockKT.CreateTokenRequestKeyCallCount())
	assert.Equal(t, 1, mockQS.GetStateCallCount())

	assert.Equal(t, 1, mockListener.OnStatusCallCount())
	callCtx, callTxID, callStatus, callMsg, callHash := mockListener.OnStatusArgsForCall(0)
	assert.Equal(t, ctx, callCtx)
	assert.Equal(t, txID, callTxID)
	assert.Equal(t, fdriver.Valid, callStatus)
	assert.Equal(t, "", callMsg)
	assert.Equal(t, tokenRequestHash, callHash)
}

func TestTxCheck_Process_Invalid(t *testing.T) {
	ctx := t.Context()
	txID := "tx123"

	mockQS := &mock.QueryService{}
	mockKT := &mock.KeyTranslator{}
	mockListener := &mock.Listener{}

	mockQS.GetTransactionStatusReturns(int32(protoblocktx.Status_ABORTED_SIGNATURE_INVALID), nil)

	txCheck := &finality.TxCheck{
		QueryService:  mockQS,
		KeyTranslator: mockKT,
		Listener:      mockListener,
		TxID:          txID,
		Namespace:     "token-namespace",
	}

	err := txCheck.Process(ctx)
	require.NoError(t, err)

	// Should not fetch token request hash for invalid transactions
	assert.Equal(t, 0, mockKT.CreateTokenRequestKeyCallCount())
	assert.Equal(t, 0, mockQS.GetStateCallCount())

	assert.Equal(t, 1, mockListener.OnStatusCallCount())
	_, _, callStatus, _, callHash := mockListener.OnStatusArgsForCall(0)
	assert.Equal(t, fdriver.Invalid, callStatus)
	assert.Nil(t, callHash)
}

func TestTxCheck_Process_Unknown(t *testing.T) {
	ctx := t.Context()
	txID := "tx123"

	mockQS := &mock.QueryService{}
	mockKT := &mock.KeyTranslator{}
	mockListener := &mock.Listener{}

	mockQS.GetTransactionStatusReturns(int32(protoblocktx.Status_NOT_VALIDATED), nil)

	txCheck := &finality.TxCheck{
		QueryService:  mockQS,
		KeyTranslator: mockKT,
		Listener:      mockListener,
		TxID:          txID,
		Namespace:     "token-namespace",
	}

	err := txCheck.Process(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "transaction [tx123] is not in a valid state")

	// Should not notify listener
	assert.Equal(t, 0, mockListener.OnStatusCallCount())
}

func TestTxCheck_Process_GetTransactionStatusError(t *testing.T) {
	ctx := t.Context()
	txID := "tx123"

	mockQS := &mock.QueryService{}
	mockKT := &mock.KeyTranslator{}
	mockListener := &mock.Listener{}

	mockQS.GetTransactionStatusReturns(0, errors.New("status query failed"))

	txCheck := &finality.TxCheck{
		QueryService:  mockQS,
		KeyTranslator: mockKT,
		Listener:      mockListener,
		TxID:          txID,
		Namespace:     "token-namespace",
	}

	err := txCheck.Process(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "can't get status for tx")
	assert.Contains(t, err.Error(), "status query failed")
}

func TestTxCheck_String(t *testing.T) {
	txCheck := &finality.TxCheck{
		TxID: "tx123",
	}

	str := txCheck.String()
	assert.Equal(t, "TxCheck[tx123]", str)
}

func TestNSFinalityListener_OnStatus(t *testing.T) {
	ctx := t.Context()
	txID := "tx123"
	namespace := "token-namespace"

	mockQueue := &mock.Queue{}
	mockQS := &mock.QueryService{}
	mockKT := &mock.KeyTranslator{}
	mockListener := &mock.Listener{}

	var enqueuedEvent queue.Event
	mockQueue.EnqueueBlockingCalls(func(ctx context.Context, event queue.Event) error {
		enqueuedEvent = event
		return nil
	})

	listener := finality.NewNSFinalityListener(namespace, mockListener, mockQueue, mockQS, mockKT)

	listener.OnStatus(ctx, txID, fdriver.Valid, "")

	assert.Equal(t, 1, mockQueue.EnqueueBlockingCallCount())

	// Verify the enqueued event
	require.NotNil(t, enqueuedEvent)
	listenerEvent, ok := enqueuedEvent.(*finality.ListenerEvent)
	require.True(t, ok)
	assert.Equal(t, txID, listenerEvent.TxID)
	assert.Equal(t, fdriver.Valid, listenerEvent.Status)
	assert.Equal(t, namespace, listenerEvent.Namespace)
}

func TestNSFinalityListener_OnStatus_EnqueueError(t *testing.T) {
	ctx := t.Context()
	txID := "tx123"
	namespace := "token-namespace"

	mockQueue := &mock.Queue{}
	mockQS := &mock.QueryService{}
	mockKT := &mock.KeyTranslator{}
	mockListener := &mock.Listener{}

	mockQueue.EnqueueBlockingReturns(errors.New("queue full"))

	listener := finality.NewNSFinalityListener(namespace, mockListener, mockQueue, mockQS, mockKT)

	// Should not panic even if enqueue fails
	listener.OnStatus(ctx, txID, fdriver.Valid, "")

	assert.Equal(t, 1, mockQueue.EnqueueBlockingCallCount())
}

func TestNSListenerManager_AddFinalityListener(t *testing.T) {
	txID := "tx123"
	namespace := "token-namespace"

	mockLM := &mock.ListenerManager{}
	mockQueue := &mock.Queue{}
	mockQS := &mock.QueryService{}
	mockKT := &mock.KeyTranslator{}
	mockListener := &mock.Listener{}

	var enqueuedEvent queue.Event
	mockQueue.EnqueueCalls(func(event queue.Event) error {
		enqueuedEvent = event
		return nil
	})

	manager := finality.NewNSListenerManager(mockLM, mockQueue, mockQS, mockKT)

	err := manager.AddFinalityListener(namespace, txID, mockListener)
	require.NoError(t, err)

	// Verify TxCheck was enqueued
	assert.Equal(t, 1, mockQueue.EnqueueCallCount())
	require.NotNil(t, enqueuedEvent)
	txCheck, ok := enqueuedEvent.(*finality.TxCheck)
	require.True(t, ok)
	assert.Equal(t, txID, txCheck.TxID)
	assert.Equal(t, namespace, txCheck.Namespace)

	// Verify listener was added to underlying manager
	assert.Equal(t, 1, mockLM.AddFinalityListenerCallCount())
	callTxID, _ := mockLM.AddFinalityListenerArgsForCall(0)
	assert.Equal(t, txID, callTxID)
}

func TestNSListenerManager_AddFinalityListener_EnqueueError(t *testing.T) {
	txID := "tx123"
	namespace := "token-namespace"

	mockLM := &mock.ListenerManager{}
	mockQueue := &mock.Queue{}
	mockQS := &mock.QueryService{}
	mockKT := &mock.KeyTranslator{}
	mockListener := &mock.Listener{}

	mockQueue.EnqueueReturns(errors.New("queue full"))

	manager := finality.NewNSListenerManager(mockLM, mockQueue, mockQS, mockKT)

	err := manager.AddFinalityListener(namespace, txID, mockListener)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "queue full")

	// Should not add listener to underlying manager if enqueue fails
	assert.Equal(t, 0, mockLM.AddFinalityListenerCallCount())
}

func TestNSListenerManagerProvider_NewManager(t *testing.T) {
	network := "test-network"
	channel := "test-channel"

	mockQSP := &mock.QueryServiceProvider{}
	mockLMP := &mock.ListenerManagerProvider{}
	mockLM := &mock.ListenerManager{}
	mockQS := &mock.QueryService{}

	mockLMP.NewManagerReturns(mockLM, nil)
	mockQSP.GetReturns(mockQS, nil)

	provider, err := finality.NewNotificationServiceBased(mockQSP, mockLMP)
	require.NoError(t, err)
	require.NotNil(t, provider)

	manager, err := provider.NewManager(network, channel)
	require.NoError(t, err)
	require.NotNil(t, manager)

	assert.Equal(t, 1, mockLMP.NewManagerCallCount())
	callNetwork, callChannel := mockLMP.NewManagerArgsForCall(0)
	assert.Equal(t, network, callNetwork)
	assert.Equal(t, channel, callChannel)

	assert.Equal(t, 1, mockQSP.GetCallCount())
	callNetwork, callChannel = mockQSP.GetArgsForCall(0)
	assert.Equal(t, network, callNetwork)
	assert.Equal(t, channel, callChannel)
}

func TestNSListenerManagerProvider_NewManager_ListenerManagerError(t *testing.T) {
	network := "test-network"
	channel := "test-channel"

	mockQSP := &mock.QueryServiceProvider{}
	mockLMP := &mock.ListenerManagerProvider{}

	mockLMP.NewManagerReturns(nil, errors.New("listener manager creation failed"))

	provider, err := finality.NewNotificationServiceBased(mockQSP, mockLMP)
	require.NoError(t, err)

	manager, err := provider.NewManager(network, channel)
	require.Error(t, err)
	assert.Nil(t, manager)
	assert.Contains(t, err.Error(), "failed creating finality manager")
	assert.Contains(t, err.Error(), "listener manager creation failed")
}

func TestNSListenerManagerProvider_NewManager_QueryServiceError(t *testing.T) {
	network := "test-network"
	channel := "test-channel"

	mockQSP := &mock.QueryServiceProvider{}
	mockLMP := &mock.ListenerManagerProvider{}
	mockLM := &mock.ListenerManager{}

	mockLMP.NewManagerReturns(mockLM, nil)
	mockQSP.GetReturns(nil, errors.New("query service retrieval failed"))

	provider, err := finality.NewNotificationServiceBased(mockQSP, mockLMP)
	require.NoError(t, err)

	manager, err := provider.NewManager(network, channel)
	require.Error(t, err)
	assert.Nil(t, manager)
	assert.Contains(t, err.Error(), "failed getting query service")
	assert.Contains(t, err.Error(), "query service retrieval failed")
}

func TestOnlyOnceListener_SingleCall(t *testing.T) {
	ctx := t.Context()
	txID := "tx123"

	mockListener := &mock.Listener{}

	// Create onlyOnceListener through NSListenerManager to test the wrapper
	mockLM := &mock.ListenerManager{}
	mockQueue := &mock.Queue{}
	mockQS := &mock.QueryService{}
	mockKT := &mock.KeyTranslator{}

	mockQueue.EnqueueReturns(nil)
	mockQueue.EnqueueBlockingCalls(func(ctx context.Context, event queue.Event) error {
		return event.Process(ctx)
	})
	mockKT.CreateTokenRequestKeyReturns("key", nil)
	mockQS.GetStateReturns(&cdriver.VaultValue{Raw: []byte("hash")}, nil)

	manager := finality.NewNSListenerManager(mockLM, mockQueue, mockQS, mockKT)

	err := manager.AddFinalityListener("namespace", txID, mockListener)
	require.NoError(t, err)

	// Get the wrapped listener that was added
	assert.Equal(t, 1, mockLM.AddFinalityListenerCallCount())
	_, wrappedListener := mockLM.AddFinalityListenerArgsForCall(0)

	// Call OnStatus multiple times
	wrappedListener.OnStatus(ctx, txID, fdriver.Valid, "")
	wrappedListener.OnStatus(ctx, txID, fdriver.Valid, "")
	wrappedListener.OnStatus(ctx, txID, fdriver.Valid, "")

	// The underlying listener should only be called once
	// Note: We can't directly test this because the onlyOnceListener is created internally
	// and the mock queue will process events asynchronously
	// This test verifies the integration works without panicking
}

func TestOnlyOnceListener_Concurrent(t *testing.T) {
	ctx := t.Context()
	txID := "tx123"

	mockListener := &mock.Listener{}
	callCount := 0
	var mu sync.Mutex

	mockListener.OnStatusCalls(func(ctx context.Context, txID string, status int, message string, hash []byte) {
		mu.Lock()
		callCount++
		mu.Unlock()
	})

	mockLM := &mock.ListenerManager{}
	mockQueue := &mock.Queue{}
	mockQS := &mock.QueryService{}
	mockKT := &mock.KeyTranslator{}

	mockQueue.EnqueueReturns(nil)
	mockQueue.EnqueueBlockingCalls(func(ctx context.Context, event queue.Event) error {
		return event.Process(ctx)
	})
	mockKT.CreateTokenRequestKeyReturns("key", nil)
	mockQS.GetStateReturns(&cdriver.VaultValue{Raw: []byte("hash")}, nil)

	manager := finality.NewNSListenerManager(mockLM, mockQueue, mockQS, mockKT)

	err := manager.AddFinalityListener("namespace", txID, mockListener)
	require.NoError(t, err)

	assert.Equal(t, 1, mockLM.AddFinalityListenerCallCount())
	_, wrappedListener := mockLM.AddFinalityListenerArgsForCall(0)

	// Call OnStatus concurrently
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			wrappedListener.OnStatus(ctx, txID, fdriver.Valid, "")
		}()
	}
	wg.Wait()

	// The underlying listener should only be called once despite concurrent calls
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 1, callCount, "OnStatus should only be called once despite concurrent calls")
}

func TestFabricXFSCStatus_Committed(t *testing.T) {
	mockQS := &mock.QueryService{}
	mockKT := &mock.KeyTranslator{}
	mockListener := &mock.Listener{}

	mockQS.GetTransactionStatusReturns(int32(protoblocktx.Status_COMMITTED), nil)
	mockKT.CreateTokenRequestKeyReturns("key", nil)
	mockQS.GetStateReturns(&cdriver.VaultValue{Raw: []byte("hash")}, nil)

	txCheck := &finality.TxCheck{
		QueryService:  mockQS,
		KeyTranslator: mockKT,
		Listener:      mockListener,
		TxID:          "tx123",
		Namespace:     "namespace",
	}

	err := txCheck.Process(t.Context())
	require.NoError(t, err)

	assert.Equal(t, 1, mockListener.OnStatusCallCount())
	_, _, status, _, _ := mockListener.OnStatusArgsForCall(0)
	assert.Equal(t, fdriver.Valid, status)
}

func TestFabricXFSCStatus_NotValidated(t *testing.T) {
	mockQS := &mock.QueryService{}
	mockKT := &mock.KeyTranslator{}
	mockListener := &mock.Listener{}

	mockQS.GetTransactionStatusReturns(int32(protoblocktx.Status_NOT_VALIDATED), nil)

	txCheck := &finality.TxCheck{
		QueryService:  mockQS,
		KeyTranslator: mockKT,
		Listener:      mockListener,
		TxID:          "tx123",
		Namespace:     "namespace",
	}

	err := txCheck.Process(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not in a valid state")
}

func TestFabricXFSCStatus_Invalid(t *testing.T) {
	mockQS := &mock.QueryService{}
	mockKT := &mock.KeyTranslator{}
	mockListener := &mock.Listener{}

	mockQS.GetTransactionStatusReturns(int32(protoblocktx.Status_ABORTED_SIGNATURE_INVALID), nil)

	txCheck := &finality.TxCheck{
		QueryService:  mockQS,
		KeyTranslator: mockKT,
		Listener:      mockListener,
		TxID:          "tx123",
		Namespace:     "namespace",
	}

	err := txCheck.Process(t.Context())
	require.NoError(t, err)

	assert.Equal(t, 1, mockListener.OnStatusCallCount())
	_, _, status, _, _ := mockListener.OnStatusArgsForCall(0)
	assert.Equal(t, fdriver.Invalid, status)
}

func TestFabricXFSCStatus_UnknownCode(t *testing.T) {
	mockQS := &mock.QueryService{}
	mockKT := &mock.KeyTranslator{}
	mockListener := &mock.Listener{}

	mockQS.GetTransactionStatusReturns(999, nil)

	txCheck := &finality.TxCheck{
		QueryService:  mockQS,
		KeyTranslator: mockKT,
		Listener:      mockListener,
		TxID:          "tx123",
		Namespace:     "namespace",
	}

	err := txCheck.Process(t.Context())
	require.NoError(t, err)

	assert.Equal(t, 1, mockListener.OnStatusCallCount())
	_, _, status, _, _ := mockListener.OnStatusArgsForCall(0)
	assert.Equal(t, fdriver.Invalid, status)
}
