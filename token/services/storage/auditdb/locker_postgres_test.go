/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package auditdb

import (
	"context"
	"database/sql"
	"sync"
	"testing"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/postgres"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func startPostgres(t *testing.T) *sql.DB {
	t.Helper()
	cfg := postgres.DefaultConfig(postgres.WithDBName("test-locker"))
	terminate, _, err := postgres.StartPostgres(t.Context(), cfg, nil)
	require.NoError(t, err)
	t.Cleanup(terminate)
	db, err := sql.Open("pgx", cfg.DataSource())
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	return db
}

func cleanTable(t *testing.T, db *sql.DB, table string) {
	t.Helper()
	_, _ = db.Exec("DROP TABLE IF EXISTS " + table)
}

func TestPostgresLocker_AcquireRelease(t *testing.T) {
	db := startPostgres(t)
	table := "test_eid_lease_ar"
	cleanTable(t, db, table)
	t.Cleanup(func() { cleanTable(t, db, table) })

	l, err := NewPostgresLocker(db, table, PostgresLockerConfig{
		TTL: 5 * time.Second, Heartbeat: 2 * time.Second,
	})
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, l.AcquireLocks(ctx, "anchor1", "alice", "bob"))
	require.NoError(t, l.AssertLocksHeld(ctx, "anchor1"))
	l.ReleaseLocks(ctx, "anchor1")

	var count int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM "+table+" WHERE anchor = $1", "anchor1").Scan(&count))
	assert.Equal(t, 0, count)
}

func TestPostgresLocker_Contention(t *testing.T) {
	db := startPostgres(t)
	table := "test_eid_lease_ct"
	cleanTable(t, db, table)
	t.Cleanup(func() { cleanTable(t, db, table) })

	cfg := PostgresLockerConfig{
		TTL: 30 * time.Second, AcquireBackoff: 50 * time.Millisecond,
		AcquireDeadline: 500 * time.Millisecond, Heartbeat: 10 * time.Second,
		Owner: "owner-1",
	}
	l1, err := NewPostgresLocker(db, table, cfg)
	require.NoError(t, err)

	cfg.Owner = "owner-2"
	l2, err := NewPostgresLocker(db, table, cfg)
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, l1.AcquireLocks(ctx, "a1", "alice"))

	err = l2.AcquireLocks(ctx, "a2", "alice")
	require.ErrorIs(t, err, ErrLockAcquireTimeout)

	l1.ReleaseLocks(ctx, "a1")

	require.NoError(t, l2.AcquireLocks(ctx, "a3", "alice"))
	l2.ReleaseLocks(ctx, "a3")
}

func TestPostgresLocker_StaleLeaseReclaim(t *testing.T) {
	db := startPostgres(t)
	table := "test_eid_lease_sl"
	cleanTable(t, db, table)
	t.Cleanup(func() { cleanTable(t, db, table) })

	cfg := PostgresLockerConfig{
		TTL: 200 * time.Millisecond, AcquireBackoff: 50 * time.Millisecond,
		AcquireDeadline: 2 * time.Second, Heartbeat: 5 * time.Second,
		Owner: "owner-1",
	}
	l1, err := NewPostgresLocker(db, table, cfg)
	require.NoError(t, err)

	cfg.Owner = "owner-2"
	l2, err := NewPostgresLocker(db, table, cfg)
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, l1.AcquireLocks(ctx, "a1", "alice"))

	time.Sleep(300 * time.Millisecond)
	require.NoError(t, l2.AcquireLocks(ctx, "a2", "alice"))
	l2.ReleaseLocks(ctx, "a2")
}

func TestPostgresLocker_AssertLocksHeld_AfterExpiry(t *testing.T) {
	db := startPostgres(t)
	table := "test_eid_lease_ah"
	cleanTable(t, db, table)
	t.Cleanup(func() { cleanTable(t, db, table) })

	l, err := NewPostgresLocker(db, table, PostgresLockerConfig{
		TTL: 200 * time.Millisecond, Heartbeat: 10 * time.Second,
		Owner: "owner-1",
	})
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, l.AcquireLocks(ctx, "a1", "alice"))

	time.Sleep(300 * time.Millisecond)
	require.ErrorIs(t, l.AssertLocksHeld(ctx, "a1"), ErrLockNotHeld)
	l.ReleaseLocks(ctx, "a1")
}

func TestPostgresLocker_SameOwnerReacquire(t *testing.T) {
	db := startPostgres(t)
	table := "test_eid_lease_so"
	cleanTable(t, db, table)
	t.Cleanup(func() { cleanTable(t, db, table) })

	l, err := NewPostgresLocker(db, table, PostgresLockerConfig{
		TTL: 5 * time.Second, Heartbeat: 2 * time.Second,
		Owner: "owner-1",
	})
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, l.AcquireLocks(ctx, "a1", "alice"))
	require.NoError(t, l.AcquireLocks(ctx, "a2", "alice"))
	l.ReleaseLocks(ctx, "a2")
}

func TestPostgresLocker_ConcurrentNonOverlapping(t *testing.T) {
	db := startPostgres(t)
	table := "test_eid_lease_cno"
	cleanTable(t, db, table)
	t.Cleanup(func() { cleanTable(t, db, table) })

	cfg := PostgresLockerConfig{
		TTL: 5 * time.Second, Heartbeat: 2 * time.Second,
		Owner: "owner-1",
	}
	l, err := NewPostgresLocker(db, table, cfg)
	require.NoError(t, err)

	ctx := context.Background()
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		assert.NoError(t, l.AcquireLocks(ctx, "a1", "alice"))
		time.Sleep(10 * time.Millisecond)
		l.ReleaseLocks(ctx, "a1")
	}()
	go func() {
		defer wg.Done()
		assert.NoError(t, l.AcquireLocks(ctx, "a2", "bob"))
		time.Sleep(10 * time.Millisecond)
		l.ReleaseLocks(ctx, "a2")
	}()
	wg.Wait()
}

func TestPostgresLocker_EmptyEIDs(t *testing.T) {
	db := startPostgres(t)
	table := "test_eid_lease_empty"
	cleanTable(t, db, table)
	t.Cleanup(func() { cleanTable(t, db, table) })

	l, err := NewPostgresLocker(db, table, PostgresLockerConfig{TTL: 5 * time.Second, Heartbeat: 2 * time.Second})
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, l.AcquireLocks(ctx, "a1"))
	l.ReleaseLocks(ctx, "a1")
}

func TestPostgresLocker_NilDB(t *testing.T) {
	_, err := NewPostgresLocker(nil, "t", PostgresLockerConfig{})
	require.Error(t, err)
}
