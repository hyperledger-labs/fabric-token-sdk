/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/auditdb/locker/dedup"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/auditdb/locker/errs"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/auditdb/locker/id"
	pgcond "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/postgres"
	q "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/query"
	qcommon "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/query/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/query/cond"
)

var logger = logging.MustGetLogger()

// Locker implements locker.Locker using a SQL lease table.
// Acquire and renew queries use Postgres-specific features (TIMESTAMPTZ,
// ON CONFLICT DO UPDATE … RETURNING, ::interval casts).
type Locker struct {
	db    *sql.DB
	table string
	cfg   Config
	ci    qcommon.CondInterpreter

	mu       sync.Mutex
	sessions map[string]*lockSession
}

type lockSession struct {
	eIDs   []string
	cancel context.CancelFunc
}

// New creates a Postgres-backed distributed Locker.
// The table is created if it does not exist. db must be a *sql.DB connected to Postgres.
func New(db *sql.DB, table string, cfg Config, replicaID id.ReplicaIDProvider) (*Locker, error) {
	if db == nil {
		return nil, errors.New("postgres locker requires a non-nil *sql.DB")
	}
	owner := ""
	if replicaID != nil {
		owner = replicaID.ID()
	}
	cfg = cfg.withDefaults(owner)
	l := &Locker{
		db:       db,
		table:    table,
		cfg:      cfg,
		ci:       pgcond.NewConditionInterpreter(),
		sessions: make(map[string]*lockSession),
	}
	if err := l.createSchema(); err != nil {
		return nil, err
	}

	return l, nil
}

func (p *Locker) createSchema() error {
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

func (p *Locker) AcquireLocks(ctx context.Context, anchor string, eIDs ...string) error {
	deduped := dedup.AndSort(eIDs)
	if len(deduped) == 0 {
		return nil
	}

	deadline := time.Now().Add(p.cfg.AcquireDeadline)
	for {
		ok, err := p.tryAcquireAll(ctx, anchor, deduped)
		if err != nil {
			return err
		}
		if ok {
			hbCtx, cancel := context.WithCancel(context.Background())
			p.mu.Lock()
			if prev, exists := p.sessions[anchor]; exists {
				prev.cancel()
			}
			p.sessions[anchor] = &lockSession{eIDs: deduped, cancel: cancel}
			p.mu.Unlock()
			go p.heartbeatLoop(hbCtx, anchor, len(deduped))

			return nil
		}
		if time.Now().After(deadline) {
			_ = p.releaseAnchor(ctx, anchor)

			return errors.Join(errs.ErrLockAcquireTimeout, errs.ErrLockContention)
		}
		select {
		case <-ctx.Done():
			_ = p.releaseAnchor(ctx, anchor)

			return ctx.Err()
		case <-time.After(p.cfg.AcquireBackoff):
		}
	}
}

func (p *Locker) tryAcquireAll(ctx context.Context, anchor string, eIDs []string) (bool, error) {
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

func (p *Locker) buildAcquireQuery(anchor string, eIDs []string) (string, []any) {
	tbl := q.Table(p.table)
	ins := q.InsertInto(p.table).
		Fields("eid", "anchor", "owner", "expires_at").
		WithBoundParams(anchor, p.cfg.Owner, p.cfg.TTL.String())
	for _, eid := range eIDs {
		ins = ins.RowValues(
			qcommon.Bind(eid),
			qcommon.Ref(1),
			qcommon.Ref(2),
			qcommon.IntervalAfterNow(3),
		)
	}

	query, args := ins.
		OnConflict([]qcommon.FieldName{"eid"},
			q.OverwriteValue("anchor"),
			q.OverwriteValue("owner"),
			q.OverwriteValue("expires_at"),
		).
		Where(cond.Or(
			cond.InPast(tbl.Field("expires_at")),
			cond.Cmp(tbl.Field("owner"), "=", q.ExcludedValue("owner")),
		)).
		Returning("eid").
		Format()

	return query, args
}

func (p *Locker) ReleaseLocks(ctx context.Context, anchor string) {
	p.stopHeartbeat(anchor)
	_ = p.releaseAnchor(ctx, anchor)
}

func (p *Locker) releaseAnchor(ctx context.Context, anchor string) error {
	query, args := q.DeleteFrom(p.table).
		Where(cond.And(cond.Eq("anchor", anchor), cond.Eq("owner", p.cfg.Owner))).
		Format(p.ci)
	_, err := p.db.ExecContext(ctx, query, args...)

	return errors.Wrap(err, "release eid leases")
}

func (p *Locker) AssertLocksHeld(ctx context.Context, anchor string) error {
	p.mu.Lock()
	s, ok := p.sessions[anchor]
	expected := 0
	if ok {
		expected = len(s.eIDs)
	}
	p.mu.Unlock()

	if !ok || expected == 0 {
		return errs.ErrLockNotHeld
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
		return errs.ErrLockNotHeld
	}

	return nil
}

func (p *Locker) heartbeatLoop(ctx context.Context, anchor string, expected int) {
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

func (p *Locker) renewLeases(ctx context.Context, anchor string, expected int) error {
	query, args := q.Update(p.table).
		SetIntervalFromNow("expires_at", p.cfg.TTL.String()).
		Where(cond.And(
			cond.Eq("anchor", anchor),
			cond.Eq("owner", p.cfg.Owner),
			cond.InFuture(qcommon.FieldName("expires_at")),
		)).
		Format(p.ci)
	res, err := p.db.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "renew eid leases")
	}
	n, err := res.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "rows affected on renew")
	}
	if int(n) != expected {
		return errs.ErrLockLost
	}

	return nil
}

func (p *Locker) stopHeartbeat(anchor string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if s, ok := p.sessions[anchor]; ok {
		s.cancel()
		delete(p.sessions, anchor)
	}
}
