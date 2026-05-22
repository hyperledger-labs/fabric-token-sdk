/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package auditdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/postgres"
	q "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/query"
	qcommon "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/query/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/query/cond"
)

const defaultTTL = 30 * time.Second
const defaultAcquireBackoff = 100 * time.Millisecond
const defaultAcquireDeadline = time.Minute
const defaultHeartbeat = 10 * time.Second

// PostgresLockerConfig holds Postgres lease-table locking settings.
type PostgresLockerConfig struct {
	// TTL is the lease duration for each EID lock row.
	TTL time.Duration
	// AcquireBackoff is the wait between retry attempts when a lock is contended.
	AcquireBackoff time.Duration
	// AcquireDeadline is the total time allowed to acquire all EID locks.
	AcquireDeadline time.Duration
	// Heartbeat is the interval at which held leases are renewed (~TTL/3).
	Heartbeat time.Duration
	// Owner identifies this replica. Generated at startup if empty.
	Owner string
}

func (c PostgresLockerConfig) withDefaults() PostgresLockerConfig {
	if c.TTL <= 0 {
		c.TTL = defaultTTL
	}
	if c.AcquireBackoff <= 0 {
		c.AcquireBackoff = defaultAcquireBackoff
	}
	if c.AcquireDeadline <= 0 {
		c.AcquireDeadline = defaultAcquireDeadline
	}
	if c.Heartbeat <= 0 {
		c.Heartbeat = defaultHeartbeat
	}
	if c.Owner == "" {
		c.Owner = generateOwnerID()
	}

	return c
}

// postgresLocker implements Locker using a SQL lease table.
// The acquire and renew queries use Postgres-specific features (TIMESTAMPTZ,
// ON CONFLICT DO UPDATE … RETURNING, ::interval casts) so the implementation
// is currently tied to PostgreSQL. Simple queries (delete, select) go through
// the project's query builder for consistency and portability.
type postgresLocker struct {
	db    *sql.DB
	table string
	cfg   PostgresLockerConfig
	ci    qcommon.CondInterpreter

	mu       sync.Mutex
	sessions map[string]*lockSession
}

type lockSession struct {
	eIDs   []string
	cancel context.CancelFunc
}

// NewPostgresLocker creates a Postgres-backed distributed Locker.
// The table is created if it does not exist. db must be a *sql.DB connected to Postgres.
func NewPostgresLocker(db *sql.DB, table string, cfg PostgresLockerConfig) (Locker, error) {
	if db == nil {
		return nil, errors.New("postgres locker requires a non-nil *sql.DB")
	}
	cfg = cfg.withDefaults()
	l := &postgresLocker{
		db:       db,
		table:    table,
		cfg:      cfg,
		ci:       postgres.NewConditionInterpreter(),
		sessions: make(map[string]*lockSession),
	}
	if err := l.createSchema(); err != nil {
		return nil, err
	}

	return l, nil
}

func (p *postgresLocker) createSchema() error {
	// #nosec G201 -- table name is configuration-driven, not user input; DDL has no query-builder support.
	schema := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			eid         TEXT        PRIMARY KEY,
			anchor      TEXT        NOT NULL,
			owner       TEXT        NOT NULL,
			expires_at  TIMESTAMPTZ NOT NULL
		);
		CREATE INDEX IF NOT EXISTS %s_anchor_idx     ON %s (anchor);
		CREATE INDEX IF NOT EXISTS %s_expires_at_idx ON %s (expires_at);
	`, p.table, p.table, p.table, p.table, p.table)
	_, err := p.db.Exec(schema)

	return errors.Wrap(err, "failed to create auditor eid lease table")
}

// AcquireLocks tries to acquire all EID leases in a single SQL statement
// (adecaro review: "try to acquire all the locks with a single SQL query").
// On contention it retries with backoff until AcquireDeadline.
func (p *postgresLocker) AcquireLocks(ctx context.Context, anchor string, eIDs ...string) error {
	dedup := deduplicateAndSort(eIDs)
	if len(dedup) == 0 {
		return nil
	}

	deadline := time.Now().Add(p.cfg.AcquireDeadline)
	for {
		ok, err := p.tryAcquireAll(ctx, anchor, dedup)
		if err != nil {
			return err
		}
		if ok {
			hbCtx, cancel := context.WithCancel(context.Background())
			p.mu.Lock()
			if prev, exists := p.sessions[anchor]; exists {
				prev.cancel()
			}
			p.sessions[anchor] = &lockSession{eIDs: dedup, cancel: cancel}
			p.mu.Unlock()
			go p.heartbeatLoop(hbCtx, anchor, len(dedup))

			return nil
		}
		if time.Now().After(deadline) {
			_ = p.releaseAnchor(ctx, anchor)

			return errors.Join(ErrLockAcquireTimeout, ErrLockContention)
		}
		select {
		case <-ctx.Done():
			_ = p.releaseAnchor(ctx, anchor)

			return ctx.Err()
		case <-time.After(p.cfg.AcquireBackoff):
		}
	}
}

// tryAcquireAll attempts a single atomic INSERT...ON CONFLICT for all EIDs.
// Returns true only when every EID was acquired.
func (p *postgresLocker) tryAcquireAll(ctx context.Context, anchor string, eIDs []string) (bool, error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return false, errors.Wrap(err, "begin eid lock tx")
	}
	defer func() { _ = tx.Rollback() }()

	query, args := p.buildAcquireQuery(anchor, eIDs)
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return false, errors.Wrap(err, "acquire eid leases")
	}
	defer func() { _ = rows.Close() }()

	acquired := 0
	for rows.Next() {
		var eid string
		if err := rows.Scan(&eid); err != nil {
			return false, errors.Wrap(err, "scan acquired eid")
		}
		acquired++
	}
	if err := rows.Err(); err != nil {
		return false, errors.Wrap(err, "iterate acquired eids")
	}
	if acquired != len(eIDs) {
		return false, nil
	}
	if err := tx.Commit(); err != nil {
		return false, errors.Wrap(err, "commit eid lock tx")
	}

	return true, nil
}

// buildAcquireQuery builds a single INSERT...VALUES...ON CONFLICT for all EIDs.
// Each EID becomes a ($N, $1, $2, NOW()+$3::interval) row; $1=anchor, $2=owner, $3=ttl.
//
// This query uses Postgres-specific features (ON CONFLICT DO UPDATE … WHERE …
// RETURNING, ::interval cast) that the project's query builder does not support.
func (p *postgresLocker) buildAcquireQuery(anchor string, eIDs []string) (string, []any) {
	args := make([]any, 0, len(eIDs)+3)
	args = append(args, anchor, p.cfg.Owner, p.cfg.TTL.String())

	valueRows := make([]string, len(eIDs))
	for i, eid := range eIDs {
		pos := len(args) + 1
		valueRows[i] = fmt.Sprintf("($%d, $1, $2, NOW() + $3::interval)", pos)
		args = append(args, eid)
	}

	// #nosec G201 -- table name is configuration-driven, not user input.
	query := fmt.Sprintf(`
INSERT INTO %s (eid, anchor, owner, expires_at)
VALUES %s
ON CONFLICT (eid) DO UPDATE
  SET anchor     = EXCLUDED.anchor,
      owner      = EXCLUDED.owner,
      expires_at = EXCLUDED.expires_at
WHERE %s.expires_at <= NOW()
   OR %s.owner = EXCLUDED.owner
RETURNING eid`, p.table, strings.Join(valueRows, ", "), p.table, p.table)

	return query, args
}

func (p *postgresLocker) ReleaseLocks(ctx context.Context, anchor string) {
	p.stopHeartbeat(anchor)
	_ = p.releaseAnchor(ctx, anchor)
}

func (p *postgresLocker) releaseAnchor(ctx context.Context, anchor string) error {
	query, args := q.DeleteFrom(p.table).
		Where(cond.And(cond.Eq("anchor", anchor), cond.Eq("owner", p.cfg.Owner))).
		Format(p.ci)
	_, err := p.db.ExecContext(ctx, query, args...)

	return errors.Wrap(err, "release eid leases")
}

// AssertLocksHeld verifies this replica still holds all leases for the anchor.
// Called before writing to the DB (adecaro review: "A replica should make
// sure it still has the lock while writing to the DB").
func (p *postgresLocker) AssertLocksHeld(ctx context.Context, anchor string) error {
	p.mu.Lock()
	s, ok := p.sessions[anchor]
	expected := 0
	if ok {
		expected = len(s.eIDs)
	}
	p.mu.Unlock()

	if !ok || expected == 0 {
		return ErrLockNotHeld
	}

	var held int
	query, args := q.Select().
		FieldsByName("COUNT(*)").
		From(q.Table(p.table)).
		Where(cond.And(
			cond.Eq("anchor", anchor),
			cond.Eq("owner", p.cfg.Owner),
			cond.InFuture(qcommon.FieldName("expires_at")),
		)).
		Format(p.ci)
	if err := p.db.QueryRowContext(ctx, query, args...).Scan(&held); err != nil {
		return errors.Wrap(err, "assert eid leases held")
	}
	if held != expected {
		return ErrLockNotHeld
	}

	return nil
}

// heartbeatLoop renews leases at the configured interval until cancelled.
func (p *postgresLocker) heartbeatLoop(ctx context.Context, anchor string, expected int) {
	ticker := time.NewTicker(p.cfg.Heartbeat)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := p.renewLeases(ctx, anchor, expected); err != nil {
				logger.Warnf("eid lease heartbeat failed for [%s]: %v", anchor, err)

				return
			}
		}
	}
}

func (p *postgresLocker) renewLeases(ctx context.Context, anchor string, expected int) error {
	// Uses Postgres-specific ::interval cast in SET; the query builder does not
	// support computed SET expressions so raw SQL is required here.
	// #nosec G201 -- table name is configuration-driven, not user input.
	query := fmt.Sprintf(
		`UPDATE %s SET expires_at = NOW() + $1::interval
WHERE anchor = $2 AND owner = $3 AND expires_at > NOW()`, p.table)
	res, err := p.db.ExecContext(ctx, query, p.cfg.TTL.String(), anchor, p.cfg.Owner)
	if err != nil {
		return errors.Wrap(err, "renew eid leases")
	}
	n, err := res.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "rows affected on renew")
	}
	if int(n) != expected {
		return ErrLockLost
	}

	return nil
}

func (p *postgresLocker) stopHeartbeat(anchor string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if s, ok := p.sessions[anchor]; ok {
		s.cancel()
		delete(p.sessions, anchor)
	}
}

func generateOwnerID() string {
	id, err := generateUUID()
	if err != nil {
		return fmt.Sprintf("auditor-%d", time.Now().UnixNano())
	}

	return "auditor-" + id
}
