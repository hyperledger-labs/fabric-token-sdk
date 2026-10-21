/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package selector

import (
	"testing"

	"github.com/LFDT-Panurus/panurus/token/services/selector/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	pgHarnessWallet = "wallet0"
	pgHarnessTokens = 1000
)

func uniformAmounts(n int, v uint64) []uint64 {
	out := make([]uint64, n)
	for i := range out {
		out[i] = v
	}

	return out
}

func pgHarnessFixture(t *testing.T, cfg pgConfig) *pgFixture {
	t.Helper()
	env := startPgEnv(t, t.Context())
	fx := newPgFixture(t, env, cfg)
	t.Cleanup(fx.close)
	fx.seedWallet(t, pgHarnessWallet, uniformAmounts(pgHarnessTokens, 1), "seed")
	fx.requireSpendable(t, pgHarnessWallet, pgHarnessTokens)

	return fx
}

// TestBenchPostgresBatchLockingIsReached checks that the default configuration claims a Select's
// tokens through LockBatch (about one round trip), and that singleLock really forces one round
// trip per token. Without it the "single" and "batch" benchmark names could silently measure the
// same path.
func TestBenchPostgresBatchLockingIsReached(t *testing.T) {
	for _, tc := range []struct {
		name       string
		singleLock bool
		check      func(t *testing.T, roundTrips int64)
	}{
		{"batch", false, func(t *testing.T, rt int64) {
			t.Helper()
			assert.LessOrEqual(t, rt, int64(3), "a 100-token Select should take a handful of round trips")
		}},
		{"single", true, func(t *testing.T, rt int64) {
			t.Helper()
			assert.GreaterOrEqual(t, rt, int64(100), "a 100-token Select should lock one token per round trip")
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := baselinePgConfig()
			cfg.singleLock = tc.singleLock
			fx := pgHarnessFixture(t, cfg)

			txID := fx.nextTxID()
			sel, err := fx.replicas[0].mgr.NewSelector(txID)
			require.NoError(t, err)
			ids, _, err := sel.Select(t.Context(), &testutils.TokenFilter{WalletID: pgHarnessWallet}, testutils.SelectQuantity, testutils.TokenType)
			require.NoError(t, err)
			require.Len(t, ids, 100)

			rt, _ := fx.pgLockCounters()
			tc.check(t, rt)
			require.NoError(t, fx.replicas[0].mgr.Close(txID))
			require.NoError(t, fx.replicas[0].mgr.Unlock(t.Context(), txID))
		})
	}
}

// TestBenchPostgresUnlockReleasesLocks checks the abort path of the benchmark loop leaves nothing
// behind, for every sweep point: leftover locks would make every later Select scan past them.
func TestBenchPostgresUnlockReleasesLocks(t *testing.T) {
	for _, cfg := range pgConfigs() {
		if cfg.driver == "simple" {
			continue // in-memory locks: nothing in the database to list
		}
		t.Run(cfg.name(), func(t *testing.T) {
			fx := pgHarnessFixture(t, cfg)
			mgr := fx.replicas[0].mgr

			txID := fx.nextTxID()
			sel, err := mgr.NewSelector(txID)
			require.NoError(t, err)
			_, _, err = sel.Select(t.Context(), &testutils.TokenFilter{WalletID: pgHarnessWallet}, testutils.SelectQuantity, testutils.TokenType)
			require.NoError(t, err)

			locks, err := fx.listLocks(t.Context())
			require.NoError(t, err)
			require.NotEmpty(t, locks, "Select should have locked tokens")

			require.NoError(t, mgr.Close(txID))
			require.NoError(t, mgr.Unlock(t.Context(), txID))
			locks, err = fx.listLocks(t.Context())
			require.NoError(t, err)
			assert.Empty(t, locks)
		})
	}
}

// TestBenchPostgresSimpleDriverSelectsAndUnlocks checks the simple driver, wired through its own
// service, selects from the Postgres token store and releases its in-memory locks.
func TestBenchPostgresSimpleDriverSelectsAndUnlocks(t *testing.T) {
	cfg := baselinePgConfig()
	cfg.driver = "simple"
	fx := pgHarnessFixture(t, cfg)
	mgr := fx.replicas[0].mgr
	filter := &testutils.TokenFilter{WalletID: pgHarnessWallet}

	txID := fx.nextTxID()
	sel, err := mgr.NewSelector(txID)
	require.NoError(t, err)
	ids, _, err := sel.Select(t.Context(), filter, testutils.SelectQuantity, testutils.TokenType)
	require.NoError(t, err)
	require.Len(t, ids, 100)

	// A second transaction cannot take the same tokens while the first holds them.
	other, err := mgr.NewSelector(fx.nextTxID())
	require.NoError(t, err)
	ids2, _, err := other.Select(t.Context(), filter, testutils.SelectQuantity, testutils.TokenType)
	require.NoError(t, err)
	held := map[string]bool{}
	for _, id := range ids {
		held[id.String()] = true
	}
	for _, id := range ids2 {
		assert.False(t, held[id.String()], "token %s selected twice", id)
	}

	require.NoError(t, mgr.Unlock(t.Context(), txID))
	again, err := mgr.NewSelector(fx.nextTxID())
	require.NoError(t, err)
	_, _, err = again.Select(t.Context(), filter, testutils.SelectQuantity, testutils.TokenType)
	require.NoError(t, err)
}

// TestBenchPostgresSettleKeepsBalance checks the settle step conserves the wallet's balance and
// leaves nothing locked once the selection is released.
func TestBenchPostgresSettleKeepsBalance(t *testing.T) {
	fx := pgHarnessFixture(t, baselinePgConfig())
	mgr := fx.replicas[0].mgr

	before, err := fx.replicas[0].tokens.Balance(t.Context(), pgHarnessWallet, testutils.TokenType)
	require.NoError(t, err)

	txID := fx.nextTxID()
	sel, err := mgr.NewSelector(txID)
	require.NoError(t, err)
	ids, total, err := sel.Select(t.Context(), &testutils.TokenFilter{WalletID: pgHarnessWallet}, testutils.SelectQuantity, testutils.TokenType)
	require.NoError(t, err)
	require.NoError(t, fx.settle(t.Context(), 0, pgHarnessWallet, txID, ids, total))
	require.NoError(t, mgr.Close(txID))
	require.NoError(t, mgr.Unlock(t.Context(), txID))

	after, err := fx.replicas[0].tokens.Balance(t.Context(), pgHarnessWallet, testutils.TokenType)
	require.NoError(t, err)
	assert.Zero(t, before.Cmp(after), "balance changed from %s to %s", before, after)
	locks, err := fx.listLocks(t.Context())
	require.NoError(t, err)
	assert.Empty(t, locks)
}
