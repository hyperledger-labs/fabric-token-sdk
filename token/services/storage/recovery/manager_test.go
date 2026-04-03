/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package recovery_test

import (
	"testing"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/recovery"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/recovery/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/ttxdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	logger := logging.MustGetLogger()
	mockDB := &mock.Storage{}
	mockHandler := &mock.Handler{}
	config := recovery.Config{
		Enabled:        true,
		TTL:            30 * time.Second,
		ScanInterval:   30 * time.Second,
		BatchSize:      100,
		WorkerCount:    1,
		LeaseDuration:  30 * time.Second,
		AdvisoryLockID: 1,
		InstanceID:     "test-instance",
	}

	manager := recovery.NewManager(
		logger,
		mockDB,
		mockHandler,
		config,
	)

	require.NotNil(t, manager)
}

func TestManager_StartStop(t *testing.T) {
	logger := logging.MustGetLogger()
	mockDB := &mock.Storage{}
	mockHandler := &mock.Handler{}
	config := recovery.Config{
		Enabled:        true,
		TTL:            100 * time.Millisecond,
		ScanInterval:   100 * time.Millisecond,
		BatchSize:      100,
		WorkerCount:    1,
		LeaseDuration:  time.Second,
		AdvisoryLockID: 1,
		InstanceID:     "test-instance",
	}

	leadership := &mock.Leadership{}
	leadership.CloseReturns(nil)
	mockDB.AcquireRecoveryLeadershipReturns(leadership, true, nil)
	mockDB.ClaimPendingTransactionsReturns([]*ttxdb.TransactionRecord{}, nil)

	manager := recovery.NewManager(
		logger,
		mockDB,
		mockHandler,
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

	assert.GreaterOrEqual(t, mockDB.AcquireRecoveryLeadershipCallCount(), 1)
	assert.GreaterOrEqual(t, mockDB.ClaimPendingTransactionsCallCount(), 1)
}

func TestManager_RecoverTransaction(t *testing.T) {
	logger := logging.MustGetLogger()
	mockDB := &mock.Storage{}
	mockHandler := &mock.Handler{}
	config := recovery.Config{
		Enabled:        true,
		TTL:            100 * time.Millisecond,
		ScanInterval:   100 * time.Millisecond,
		BatchSize:      100,
		WorkerCount:    1,
		LeaseDuration:  time.Second,
		AdvisoryLockID: 1,
		InstanceID:     "test-instance",
	}

	// Create a pending transaction record
	txRecord := &ttxdb.TransactionRecord{
		TxID:   "tx123",
		Status: storage.Pending,
	}

	leadership1 := &mock.Leadership{}
	leadership1.CloseReturns(nil)
	leadership2 := &mock.Leadership{}
	leadership2.CloseReturns(nil)

	mockDB.AcquireRecoveryLeadershipReturnsOnCall(0, leadership1, true, nil)
	mockDB.AcquireRecoveryLeadershipReturns(leadership2, true, nil)
	mockDB.ClaimPendingTransactionsReturnsOnCall(0, []*ttxdb.TransactionRecord{txRecord}, nil)
	mockDB.ClaimPendingTransactionsReturns([]*ttxdb.TransactionRecord{}, nil)
	mockHandler.RecoverReturns(nil)
	mockDB.ReleaseRecoveryClaimReturns(nil)

	manager := recovery.NewManager(
		logger,
		mockDB,
		mockHandler,
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
	assert.GreaterOrEqual(t, mockHandler.RecoverCallCount(), 1)
	_, txID := mockHandler.RecoverArgsForCall(0)
	assert.Equal(t, "tx123", txID)
	assert.GreaterOrEqual(t, mockDB.ReleaseRecoveryClaimCallCount(), 1)
}

func TestManager_SkipAlreadyRecovered(t *testing.T) {
	logger := logging.MustGetLogger()
	mockDB := &mock.Storage{}
	mockHandler := &mock.Handler{}
	config := recovery.Config{
		Enabled:        true,
		TTL:            50 * time.Millisecond,
		ScanInterval:   50 * time.Millisecond,
		BatchSize:      100,
		WorkerCount:    1,
		LeaseDuration:  time.Second,
		AdvisoryLockID: 1,
		InstanceID:     "test-instance",
	}

	// Create a pending transaction record
	txRecord := &ttxdb.TransactionRecord{
		TxID:   "tx456",
		Status: storage.Pending,
	}

	leadershipA := &mock.Leadership{}
	leadershipA.CloseReturns(nil)
	leadershipB := &mock.Leadership{}
	leadershipB.CloseReturns(nil)
	leadershipC := &mock.Leadership{}
	leadershipC.CloseReturns(nil)

	mockDB.AcquireRecoveryLeadershipReturnsOnCall(0, leadershipA, true, nil)
	mockDB.AcquireRecoveryLeadershipReturnsOnCall(1, leadershipB, true, nil)
	mockDB.AcquireRecoveryLeadershipReturns(leadershipC, true, nil)
	mockDB.ClaimPendingTransactionsReturnsOnCall(0, []*ttxdb.TransactionRecord{txRecord}, nil)
	mockDB.ClaimPendingTransactionsReturns([]*ttxdb.TransactionRecord{}, nil)
	mockHandler.RecoverReturns(nil)
	mockDB.ReleaseRecoveryClaimReturns(nil)

	manager := recovery.NewManager(
		logger,
		mockDB,
		mockHandler,
		config,
	)

	// Start the manager
	err := manager.Start()
	require.NoError(t, err)

	// Wait for multiple scan cycles
	time.Sleep(200 * time.Millisecond)

	// Stop the manager
	_ = manager.Stop()

	// Verify Recover was called only once (not on subsequent scans because claim was released)
	assert.Equal(t, 1, mockHandler.RecoverCallCount())
}

func TestDefaultConfig(t *testing.T) {
	config := recovery.DefaultConfig()

	assert.True(t, config.Enabled, "Recovery should be enabled by default")
	assert.Equal(t, 30*time.Second, config.TTL)
	assert.Equal(t, 5*time.Second, config.ScanInterval)
	assert.Equal(t, 100, config.BatchSize)
	assert.Equal(t, 4, config.WorkerCount)
	assert.Equal(t, 30*time.Second, config.LeaseDuration)
}
