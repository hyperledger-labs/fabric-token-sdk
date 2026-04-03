/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package recovery_test

import (
	"context"
	"testing"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/recovery"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/ttxdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockTTXDatabase is a mock implementation of recovery.TTXDatabase
type MockTTXDatabase struct {
	mock.Mock
}

func (m *MockTTXDatabase) QueryPendingTransactions(ctx context.Context, olderThan time.Duration) ([]*ttxdb.TransactionRecord, error) {
	args := m.Called(ctx, olderThan)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).([]*ttxdb.TransactionRecord), args.Error(1)
}

func (m *MockTTXDatabase) GetTokenRequest(ctx context.Context, txID string) ([]byte, error) {
	args := m.Called(ctx, txID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockTTXDatabase) SetStatus(ctx context.Context, txID string, status storage.TxStatus, message string) error {
	args := m.Called(ctx, txID, status, message)

	return args.Error(0)
}

// MockNetwork is a mock implementation of dep.Network
type MockNetwork struct {
	mock.Mock
}

func (m *MockNetwork) AddFinalityListener(namespace string, txID string, listener network.FinalityListener) error {
	args := m.Called(namespace, txID, listener)

	return args.Error(0)
}

func (m *MockNetwork) NewEnvelope() *network.Envelope {
	args := m.Called()

	return args.Get(0).(*network.Envelope)
}

func (m *MockNetwork) AnonymousIdentity() (view.Identity, error) {
	args := m.Called()

	return args.Get(0).(view.Identity), args.Error(1)
}

func (m *MockNetwork) LocalMembership() *network.LocalMembership {
	args := m.Called()

	return args.Get(0).(*network.LocalMembership)
}

func (m *MockNetwork) ComputeTxID(n *network.TxID) string {
	args := m.Called(n)

	return args.String(0)
}

// MockFinalityListenerFactory is a mock implementation of recovery.FinalityListenerFactory
type MockFinalityListenerFactory struct {
	mock.Mock
}

func (m *MockFinalityListenerFactory) NewFinalityListener(txID string) (network.FinalityListener, error) {
	args := m.Called(txID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(network.FinalityListener), args.Error(1)
}

// MockFinalityListener is a mock implementation of network.FinalityListener
type MockFinalityListener struct {
	mock.Mock
}

func (m *MockFinalityListener) OnStatus(ctx context.Context, txID string, status int, message string, tokenRequestHash []byte) {
	m.Called(ctx, txID, status, message, tokenRequestHash)
}

func (m *MockFinalityListener) OnError(ctx context.Context, txID string, err error) {
	m.Called(ctx, txID, err)
}

func TestNewManager(t *testing.T) {
	logger := logging.MustGetLogger()
	mockDB := &MockTTXDatabase{}
	mockNetwork := &MockNetwork{}
	mockFactory := &MockFinalityListenerFactory{}
	config := recovery.Config{
		Enabled:      true,
		TTL:          30 * time.Second,
		ScanInterval: 30 * time.Second,
	}

	manager := recovery.NewManager(
		logger,
		mockDB,
		mockNetwork,
		"testns",
		mockFactory,
		config,
	)

	require.NotNil(t, manager)
}

func TestManager_StartStop(t *testing.T) {
	logger := logging.MustGetLogger()
	mockDB := &MockTTXDatabase{}
	mockNetwork := &MockNetwork{}
	mockFactory := &MockFinalityListenerFactory{}
	config := recovery.Config{
		Enabled:      true,
		TTL:          100 * time.Millisecond,
		ScanInterval: 100 * time.Millisecond,
	}

	// Setup mock to return empty list of pending transactions
	mockDB.On("QueryPendingTransactions", mock.Anything, mock.Anything).Return([]*ttxdb.TransactionRecord{}, nil)

	manager := recovery.NewManager(
		logger,
		mockDB,
		mockNetwork,
		"testns",
		mockFactory,
		config,
	)

	// Start the manager
	err := manager.Start()
	require.NoError(t, err)

	// Starting again should return error
	err = manager.Start()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already started")

	// Wait a bit to let it scan at least once
	time.Sleep(150 * time.Millisecond)

	// Stop the manager
	_ = manager.Stop()

	// Verify QueryPendingTransactions was called at least once
	mockDB.AssertCalled(t, "QueryPendingTransactions", mock.Anything, mock.Anything)
}

func TestManager_RecoverTransaction(t *testing.T) {
	logger := logging.MustGetLogger()
	mockDB := &MockTTXDatabase{}
	mockNetwork := &MockNetwork{}
	mockFactory := &MockFinalityListenerFactory{}
	config := recovery.Config{
		Enabled:      true,
		TTL:          100 * time.Millisecond,
		ScanInterval: 100 * time.Millisecond,
	}

	// Create a pending transaction record
	txRecord := &ttxdb.TransactionRecord{
		TxID:   "tx123",
		Status: storage.Pending,
	}

	// Setup mocks
	mockDB.On("QueryPendingTransactions", mock.Anything, mock.Anything).Return([]*ttxdb.TransactionRecord{txRecord}, nil).Once()
	mockDB.On("QueryPendingTransactions", mock.Anything, mock.Anything).Return([]*ttxdb.TransactionRecord{}, nil) // Subsequent calls return empty
	mockFactory.On("NewFinalityListener", "tx123").Return(&MockFinalityListener{}, nil)
	mockNetwork.On("AddFinalityListener", "testns", "tx123", mock.Anything).Return(nil)

	manager := recovery.NewManager(
		logger,
		mockDB,
		mockNetwork,
		"testns",
		mockFactory,
		config,
	)

	// Start the manager
	err := manager.Start()
	require.NoError(t, err)

	// Wait for recovery to happen
	time.Sleep(200 * time.Millisecond)

	// Stop the manager
	_ = manager.Stop()

	// Verify the transaction was recovered
	mockNetwork.AssertCalled(t, "AddFinalityListener", "testns", "tx123", mock.Anything)
}

func TestManager_SkipAlreadyRecovered(t *testing.T) {
	logger := logging.MustGetLogger()
	mockDB := &MockTTXDatabase{}
	mockNetwork := &MockNetwork{}
	mockFactory := &MockFinalityListenerFactory{}
	config := recovery.Config{
		Enabled:      true,
		TTL:          50 * time.Millisecond,
		ScanInterval: 50 * time.Millisecond,
	}

	// Create a pending transaction record
	txRecord := &ttxdb.TransactionRecord{
		TxID:   "tx456",
		Status: storage.Pending,
	}

	// Setup mocks - return the same transaction multiple times
	mockDB.On("QueryPendingTransactions", mock.Anything, mock.Anything).Return([]*ttxdb.TransactionRecord{txRecord}, nil)
	mockFactory.On("NewFinalityListener", "tx456").Return(&MockFinalityListener{}, nil)
	mockNetwork.On("AddFinalityListener", "testns", "tx456", mock.Anything).Return(nil)

	manager := recovery.NewManager(
		logger,
		mockDB,
		mockNetwork,
		"testns",
		mockFactory,
		config,
	)

	// Start the manager
	err := manager.Start()
	require.NoError(t, err)

	// Wait for multiple scan cycles
	time.Sleep(200 * time.Millisecond)

	// Stop the manager
	_ = manager.Stop()

	// Verify AddFinalityListener was called only once (not on subsequent scans)
	mockNetwork.AssertNumberOfCalls(t, "AddFinalityListener", 1)
}

func TestDefaultConfig(t *testing.T) {
	config := recovery.DefaultConfig()

	assert.False(t, config.Enabled, "Recovery should be disabled by default")
	assert.Equal(t, 30*time.Second, config.TTL)
	assert.Equal(t, 30*time.Second, config.ScanInterval)
}
