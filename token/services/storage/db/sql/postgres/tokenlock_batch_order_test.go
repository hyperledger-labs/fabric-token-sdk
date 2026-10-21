/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package postgres

import (
	"database/sql"
	"fmt"
	"math/rand/v2"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	sqlcommon "github.com/LFDT-Panurus/panurus/token/services/storage/db/sql/common"
	"github.com/LFDT-Panurus/panurus/token/token"
	common2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/common"
	fscpostgres "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/postgres"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
)

// TestTokenLockStore_LockBatch_OverlappingClaimsDoNotDeadlock claims overlapping windows from
// many goroutines, each listing its candidates in a different random order, and requires that
// no claim is aborted by Postgres' deadlock detector. Concurrent claimants really do list
// overlapping windows in different orders in production: sherdlock shuffles equal-amount
// candidates per Select. Without a fixed insert order, claims that insert the same lock keys in
// opposite orders wait for each other.
//
// The table is large (50,000 rows) so that the planner drives the claim from the candidate list
// (a nested loop) rather than from a scan of the table, which would insert in table order for
// everybody and hide the problem; the candidates are drawn from a small hot subset so that
// concurrent windows overlap heavily. deadlock_timeout is lowered through the connection string
// so that a regression fails fast instead of costing a second per deadlock.
func TestTokenLockStore_LockBatch_OverlappingClaimsDoNotDeadlock(t *testing.T) {
	const (
		tableRows  = 50_000
		hot        = 300
		window     = 100
		claimants  = 16
		perClaimer = 25
	)
	for _, strategy := range []string{sqlcommon.LockStrategyInsert, sqlcommon.LockStrategyOnConflict, sqlcommon.LockStrategySkipLocked} {
		t.Run(strategy, func(t *testing.T) {
			cfg := fscpostgres.DefaultConfig(fscpostgres.WithDBName("test-batch-order-" + strings.ToLower(strategy)))
			startPostgres(t, cfg)

			db, err := sql.Open("pgx", cfg.DataSource()+" options='-c deadlock_timeout=50ms'")
			require.NoError(t, err)
			db.SetMaxOpenConns(claimants + 2)
			t.Cleanup(func() { _ = db.Close() })

			tables, err := sqlcommon.GetTableNames("")
			require.NoError(t, err)
			tokenStore, err := sqlcommon.NewTokenStoreWithNotifier(db, db, tables, NewConditionInterpreter(), nil)
			require.NoError(t, err)
			require.NoError(t, tokenStore.CreateSchema())
			lockStore, err := newTokenLockStoreWithStrategy(&common2.RWDB{ReadDB: db, WriteDB: db}, tables, strategy)
			require.NoError(t, err)
			require.NoError(t, lockStore.CreateSchema())

			_, err = db.ExecContext(t.Context(), fmt.Sprintf(`
				INSERT INTO %s (tx_id, idx, amount, token_type, quantity, owner_raw, owner_type, owner_identity,
				                owner_wallet_id, ledger, ledger_metadata, stored_at, owner)
				SELECT 'seed-' || i, 0, 1, 'USD', '0x1', 'w'::bytea, 'idemix', ''::bytea, 'w', 'l'::bytea,
				       ''::bytea, now(), true
				FROM generate_series(0, %d) AS i`, tables.Tokens, tableRows-1))
			require.NoError(t, err)
			_, err = db.ExecContext(t.Context(), "ANALYZE "+tables.Tokens)
			require.NoError(t, err)

			var deadlocks, otherErrors atomic.Int64
			var slowest atomic.Int64
			var wg sync.WaitGroup
			for g := range claimants {
				wg.Go(func() {
					rng := rand.New(rand.NewPCG(11, uint64(g))) //nolint:gosec // g is a small non-negative index
					for r := range perClaimer {
						ids := make([]*token.ID, 0, window)
						for _, i := range rng.Perm(hot)[:window] { // random subset, random order
							ids = append(ids, &token.ID{TxId: fmt.Sprintf("seed-%d", i)})
						}
						txID := fmt.Sprintf("claimant-%d-%d", g, r)
						start := time.Now()
						_, err := lockStore.LockBatch(t.Context(), ids, txID, "w")
						slowest.Store(max(slowest.Load(), time.Since(start).Nanoseconds()))
						switch {
						case err == nil:
						case strings.Contains(err.Error(), "40P01"):
							deadlocks.Add(1)
						default:
							otherErrors.Add(1)
						}
						require.NoError(t, lockStore.UnlockByTxID(t.Context(), txID))
					}
				})
			}
			wg.Wait()

			require.Zero(t, otherErrors.Load(), "unexpected errors from LockBatch")
			require.Zero(t, deadlocks.Load(),
				"%d of %d overlapping claims were aborted by the deadlock detector (slowest claim %s)",
				deadlocks.Load(), claimants*perClaimer, time.Duration(slowest.Load()))
		})
	}
}
