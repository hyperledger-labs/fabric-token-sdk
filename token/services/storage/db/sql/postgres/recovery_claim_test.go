/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package postgres

import (
	"context"
	"math/big"
	"testing"
	"time"

	tokensdriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/driver"
	"github.com/stretchr/testify/require"
)

// TestClaimPendingTransactions_Atomic verifies that only one instance can claim transactions
func TestClaimPendingTransactions_Atomic(t *testing.T) {
	terminate, pgConnStr := startContainer(t)
	defer terminate()

	ctx := context.Background()

	// Create two transaction stores (simulating two recovery instances)
	driver1 := NewDriver(postgresCfg(pgConnStr, "claim_atomic_test"))
	driver2 := NewDriver(postgresCfg(pgConnStr, "claim_atomic_test"))

	store1Interface, err := driver1.NewOwnerTransaction("test", "claim_atomic_test")
	require.NoError(t, err)
	store1, ok := store1Interface.(*TransactionStore)
	require.True(t, ok)

	store2Interface, err := driver2.NewOwnerTransaction("test", "claim_atomic_test")
	require.NoError(t, err)
	store2, ok := store2Interface.(*TransactionStore)
	require.True(t, ok)

	// Create schema
	err = store1.CreateSchema()
	require.NoError(t, err)

	// Add test transactions
	aw, err := store1.NewTransactionStoreTransaction()
	require.NoError(t, err)

	now := time.Now().UTC()
	oldTime := now.Add(-10 * time.Minute)

	// Add 5 pending transactions
	for i := range 5 {
		txID := "tx" + string(rune('1'+i))
		err = aw.AddTokenRequest(ctx, txID, []byte("request"), nil, nil, []byte("hash"))
		require.NoError(t, err)

		err = aw.AddTransaction(ctx, tokensdriver.TransactionRecord{
			TxID:         txID,
			ActionType:   tokensdriver.Transfer,
			SenderEID:    "sender",
			RecipientEID: "recipient",
			TokenType:    "USD",
			Amount:       big.NewInt(100),
			Timestamp:    oldTime,
		})
		require.NoError(t, err)
	}

	err = aw.Commit()
	require.NoError(t, err)

	// Both instances try to claim the same transactions
	params := tokensdriver.RecoveryClaimParams{
		OlderThan:     now,
		LeaseDuration: 5 * time.Minute,
		Limit:         10,
		Owner:         "instance1",
	}

	claimed1, err := store1.ClaimPendingTransactions(ctx, params)
	require.NoError(t, err)
	require.Len(t, claimed1, 5, "Instance 1 should claim all 5 transactions")

	// Instance 2 tries to claim - should get nothing
	params.Owner = "instance2"
	claimed2, err := store2.ClaimPendingTransactions(ctx, params)
	require.NoError(t, err)
	require.Empty(t, claimed2, "Instance 2 should claim nothing (already claimed)")
}

// TestClaimPendingTransactions_Lease verifies lease expiration works
func TestClaimPendingTransactions_Lease(t *testing.T) {
	terminate, pgConnStr := startContainer(t)
	defer terminate()

	ctx := context.Background()

	driver := NewDriver(postgresCfg(pgConnStr, "claim_lease_test"))
	storeInterface, err := driver.NewOwnerTransaction("test", "claim_lease_test")
	require.NoError(t, err)
	store, ok := storeInterface.(*TransactionStore)
	require.True(t, ok)

	err = store.CreateSchema()
	require.NoError(t, err)

	// Add test transaction
	aw, err := store.NewTransactionStoreTransaction()
	require.NoError(t, err)

	now := time.Now().UTC()
	oldTime := now.Add(-10 * time.Minute)

	txID := "tx1"
	err = aw.AddTokenRequest(ctx, txID, []byte("request"), nil, nil, []byte("hash"))
	require.NoError(t, err)

	err = aw.AddTransaction(ctx, tokensdriver.TransactionRecord{
		TxID:         txID,
		ActionType:   tokensdriver.Transfer,
		SenderEID:    "sender",
		RecipientEID: "recipient",
		TokenType:    "USD",
		Amount:       big.NewInt(100),
		Timestamp:    oldTime,
	})
	require.NoError(t, err)

	err = aw.Commit()
	require.NoError(t, err)

	// Claim with very short lease
	params := tokensdriver.RecoveryClaimParams{
		OlderThan:     now,
		LeaseDuration: 1 * time.Second,
		Limit:         10,
		Owner:         "instance1",
	}

	claimed, err := store.ClaimPendingTransactions(ctx, params)
	require.NoError(t, err)
	require.Len(t, claimed, 1)

	// Try to claim again immediately with DIFFERENT owner - should get nothing
	params.Owner = "instance2"
	claimed, err = store.ClaimPendingTransactions(ctx, params)
	require.NoError(t, err)
	require.Empty(t, claimed, "Should not re-claim before lease expires")

	// Wait for lease to expire
	time.Sleep(2 * time.Second)

	// Try to claim again with same owner (idempotent) or different owner (expired)
	claimed, err = store.ClaimPendingTransactions(ctx, params)
	require.NoError(t, err)
	require.Len(t, claimed, 1, "Should re-claim after lease expires")
}

// TestClaimPendingTransactions_Idempotent verifies same owner can re-claim
func TestClaimPendingTransactions_Idempotent(t *testing.T) {
	terminate, pgConnStr := startContainer(t)
	defer terminate()

	ctx := context.Background()

	driver := NewDriver(postgresCfg(pgConnStr, "claim_idempotent_test"))
	storeInterface, err := driver.NewOwnerTransaction("test", "claim_idempotent_test")
	require.NoError(t, err)
	store, ok := storeInterface.(*TransactionStore)
	require.True(t, ok)

	err = store.CreateSchema()
	require.NoError(t, err)

	// Add test transaction
	aw, err := store.NewTransactionStoreTransaction()
	require.NoError(t, err)

	now := time.Now().UTC()
	oldTime := now.Add(-10 * time.Minute)

	txID := "tx1"
	err = aw.AddTokenRequest(ctx, txID, []byte("request"), nil, nil, []byte("hash"))
	require.NoError(t, err)

	err = aw.AddTransaction(ctx, tokensdriver.TransactionRecord{
		TxID:         txID,
		ActionType:   tokensdriver.Transfer,
		SenderEID:    "sender",
		RecipientEID: "recipient",
		TokenType:    "USD",
		Amount:       big.NewInt(100),
		Timestamp:    oldTime,
	})
	require.NoError(t, err)

	err = aw.Commit()
	require.NoError(t, err)

	// Claim transaction
	params := tokensdriver.RecoveryClaimParams{
		OlderThan:     now,
		LeaseDuration: 5 * time.Minute,
		Limit:         10,
		Owner:         "instance1",
	}

	claimed, err := store.ClaimPendingTransactions(ctx, params)
	require.NoError(t, err)
	require.Len(t, claimed, 1)

	// Same owner tries to claim again - should succeed (idempotent)
	claimed, err = store.ClaimPendingTransactions(ctx, params)
	require.NoError(t, err)
	require.Len(t, claimed, 1, "Same owner should be able to re-claim")
}

// TestClaimPendingTransactions_Limit verifies limit is respected
func TestClaimPendingTransactions_Limit(t *testing.T) {
	terminate, pgConnStr := startContainer(t)
	defer terminate()

	ctx := context.Background()

	driver := NewDriver(postgresCfg(pgConnStr, "claim_limit_test"))
	storeInterface, err := driver.NewOwnerTransaction("test", "claim_limit_test")
	require.NoError(t, err)
	store, ok := storeInterface.(*TransactionStore)
	require.True(t, ok)

	err = store.CreateSchema()
	require.NoError(t, err)

	// Add 10 test transactions
	aw, err := store.NewTransactionStoreTransaction()
	require.NoError(t, err)

	now := time.Now().UTC()
	oldTime := now.Add(-10 * time.Minute)

	for i := range 10 {
		txID := "tx" + string(rune('0'+i))
		err = aw.AddTokenRequest(ctx, txID, []byte("request"), nil, nil, []byte("hash"))
		require.NoError(t, err)

		err = aw.AddTransaction(ctx, tokensdriver.TransactionRecord{
			TxID:         txID,
			ActionType:   tokensdriver.Transfer,
			SenderEID:    "sender",
			RecipientEID: "recipient",
			TokenType:    "USD",
			Amount:       big.NewInt(100),
			Timestamp:    oldTime.Add(time.Duration(i) * time.Second),
		})
		require.NoError(t, err)
	}

	err = aw.Commit()
	require.NoError(t, err)

	// Claim with limit of 3
	params := tokensdriver.RecoveryClaimParams{
		OlderThan:     now,
		LeaseDuration: 5 * time.Minute,
		Limit:         3,
		Owner:         "instance1",
	}

	claimed, err := store.ClaimPendingTransactions(ctx, params)
	require.NoError(t, err)
	require.Len(t, claimed, 3, "Should respect limit of 3")

	// Claim again with different owner - should get next 3
	params.Owner = "instance2"
	claimed, err = store.ClaimPendingTransactions(ctx, params)
	require.NoError(t, err)
	require.Len(t, claimed, 3, "Should claim next 3 transactions")
}

// TestReleaseRecoveryClaim verifies claim release works
func TestReleaseRecoveryClaim(t *testing.T) {
	terminate, pgConnStr := startContainer(t)
	defer terminate()

	ctx := context.Background()

	driver := NewDriver(postgresCfg(pgConnStr, "release_claim_test"))
	storeInterface, err := driver.NewOwnerTransaction("test", "release_claim_test")
	require.NoError(t, err)
	store, ok := storeInterface.(*TransactionStore)
	require.True(t, ok)

	err = store.CreateSchema()
	require.NoError(t, err)

	// Add test transaction
	aw, err := store.NewTransactionStoreTransaction()
	require.NoError(t, err)

	now := time.Now().UTC()
	oldTime := now.Add(-10 * time.Minute)

	txID := "tx1"
	err = aw.AddTokenRequest(ctx, txID, []byte("request"), nil, nil, []byte("hash"))
	require.NoError(t, err)

	err = aw.AddTransaction(ctx, tokensdriver.TransactionRecord{
		TxID:         txID,
		ActionType:   tokensdriver.Transfer,
		SenderEID:    "sender",
		RecipientEID: "recipient",
		TokenType:    "USD",
		Amount:       big.NewInt(100),
		Timestamp:    oldTime,
	})
	require.NoError(t, err)

	err = aw.Commit()
	require.NoError(t, err)

	// Claim transaction
	params := tokensdriver.RecoveryClaimParams{
		OlderThan:     now,
		LeaseDuration: 5 * time.Minute,
		Limit:         10,
		Owner:         "instance1",
	}

	claimed, err := store.ClaimPendingTransactions(ctx, params)
	require.NoError(t, err)
	require.Len(t, claimed, 1)

	// Release the claim
	err = store.ReleaseRecoveryClaim(ctx, txID, "instance1", "recovery completed")
	require.NoError(t, err)

	// Different owner should now be able to claim
	params.Owner = "instance2"
	claimed, err = store.ClaimPendingTransactions(ctx, params)
	require.NoError(t, err)
	require.Len(t, claimed, 1, "Should be able to claim after release")
}

// TestReleaseRecoveryClaim_WrongOwner verifies ownership check
func TestReleaseRecoveryClaim_WrongOwner(t *testing.T) {
	terminate, pgConnStr := startContainer(t)
	defer terminate()

	ctx := context.Background()

	driver := NewDriver(postgresCfg(pgConnStr, "release_wrong_owner_test"))
	storeInterface, err := driver.NewOwnerTransaction("test", "release_wrong_owner_test")
	require.NoError(t, err)
	store, ok := storeInterface.(*TransactionStore)
	require.True(t, ok)

	err = store.CreateSchema()
	require.NoError(t, err)

	// Add test transaction
	aw, err := store.NewTransactionStoreTransaction()
	require.NoError(t, err)

	now := time.Now().UTC()
	oldTime := now.Add(-10 * time.Minute)

	txID := "tx1"
	err = aw.AddTokenRequest(ctx, txID, []byte("request"), nil, nil, []byte("hash"))
	require.NoError(t, err)

	err = aw.AddTransaction(ctx, tokensdriver.TransactionRecord{
		TxID:         txID,
		ActionType:   tokensdriver.Transfer,
		SenderEID:    "sender",
		RecipientEID: "recipient",
		TokenType:    "USD",
		Amount:       big.NewInt(100),
		Timestamp:    oldTime,
	})
	require.NoError(t, err)

	err = aw.Commit()
	require.NoError(t, err)

	// Claim transaction
	params := tokensdriver.RecoveryClaimParams{
		OlderThan:     now,
		LeaseDuration: 5 * time.Minute,
		Limit:         10,
		Owner:         "instance1",
	}

	claimed, err := store.ClaimPendingTransactions(ctx, params)
	require.NoError(t, err)
	require.Len(t, claimed, 1)

	// Try to release with wrong owner - should not release
	err = store.ReleaseRecoveryClaim(ctx, txID, "instance2", "recovery completed")
	require.NoError(t, err) // No error, but claim not released

	// Original owner should still be able to re-claim
	claimed, err = store.ClaimPendingTransactions(ctx, params)
	require.NoError(t, err)
	require.Len(t, claimed, 1, "Original owner should still have claim")
}

// TestCleanupExpiredClaims verifies expired claims are cleaned up
func TestCleanupExpiredClaims(t *testing.T) {
	terminate, pgConnStr := startContainer(t)
	defer terminate()

	ctx := context.Background()

	driver := NewDriver(postgresCfg(pgConnStr, "cleanup_test"))
	storeInterface, err := driver.NewOwnerTransaction("test", "cleanup_test")
	require.NoError(t, err)
	store, ok := storeInterface.(*TransactionStore)
	require.True(t, ok)

	err = store.CreateSchema()
	require.NoError(t, err)

	// Add test transactions
	aw, err := store.NewTransactionStoreTransaction()
	require.NoError(t, err)

	now := time.Now().UTC()
	oldTime := now.Add(-10 * time.Minute)

	for i := range 3 {
		txID := "tx" + string(rune('1'+i))
		err = aw.AddTokenRequest(ctx, txID, []byte("request"), nil, nil, []byte("hash"))
		require.NoError(t, err)

		err = aw.AddTransaction(ctx, tokensdriver.TransactionRecord{
			TxID:         txID,
			ActionType:   tokensdriver.Transfer,
			SenderEID:    "sender",
			RecipientEID: "recipient",
			TokenType:    "USD",
			Amount:       big.NewInt(100),
			Timestamp:    oldTime,
		})
		require.NoError(t, err)
	}

	err = aw.Commit()
	require.NoError(t, err)

	// Claim with very short lease
	params := tokensdriver.RecoveryClaimParams{
		OlderThan:     now,
		LeaseDuration: 1 * time.Second,
		Limit:         10,
		Owner:         "instance1",
	}

	claimed, err := store.ClaimPendingTransactions(ctx, params)
	require.NoError(t, err)
	require.Len(t, claimed, 3)

	// Wait for lease to expire
	time.Sleep(2 * time.Second)

	// Cleanup expired claims
	count, err := store.CleanupExpiredClaims(ctx)
	require.NoError(t, err)
	require.Equal(t, 3, count, "Should cleanup 3 expired claims")

	// Different owner should now be able to claim
	params.Owner = "instance2"
	claimed, err = store.ClaimPendingTransactions(ctx, params)
	require.NoError(t, err)
	require.Len(t, claimed, 3, "Should be able to claim after cleanup")
}
