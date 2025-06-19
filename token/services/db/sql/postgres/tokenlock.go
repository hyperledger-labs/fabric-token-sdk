/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections/iterators"
	common2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/common"
	common4 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/postgres"
	q "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/query"
	common3 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/query/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/query/cond"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"go.uber.org/zap/zapcore"
)

type TokenLockStore struct {
	*common.TokenLockStore

	ci common3.CondInterpreter
}

func NewTokenLockStore(dbs *common2.RWDB, tableNames common.TableNames) (*TokenLockStore, error) {
	ci := postgres.NewConditionInterpreter()
	tldb, err := common.NewTokenLockStore(dbs.ReadDB, dbs.WriteDB, tableNames, ci)
	if err != nil {
		return nil, err
	}
	return &TokenLockStore{TokenLockStore: tldb, ci: ci}, nil
}

func (db *TokenLockStore) Cleanup(ctx context.Context, leaseExpiry time.Duration) error {
	if _, err := db.logStaleLocks(ctx, leaseExpiry); err != nil {
		db.Logger.Warnf("Could not log stale locks: %v", err)
	}
	tokenLocks, tokenRequests := q.Table(db.Table.TokenLocks), q.Table(db.Table.Requests)
	query, args := common3.NewBuilder().
		WriteString("DELETE FROM ").
		WriteConditionSerializable(tokenLocks, db.ci).
		WriteString(" USING ").
		WriteConditionSerializable(tokenRequests, db.ci).
		WriteString(" WHERE ").
		WriteConditionSerializable(cond.And(
			cond.Cmp(tokenLocks.Field("consumer_tx_id"), "=", tokenRequests.Field("tx_id")),
			common.IsExpiredToken(tokenRequests, tokenLocks, leaseExpiry)), db.ci).
		Build()

	db.Logger.Debug(query)
	_, err := db.WriteDB.ExecContext(ctx, query, args...)
	if err != nil {
		db.Logger.Errorf("query failed: %s", query)
	}
	return err
}

func (db *TokenLockStore) logStaleLocks(ctx context.Context, leaseExpiry time.Duration) ([]lockEntry, error) {
	if !db.Logger.IsEnabledFor(zapcore.InfoLevel) {
		return nil, nil
	}
	tokenLocks, tokenRequests := q.Table(db.Table.TokenLocks), q.Table(db.Table.Requests)

	query, args := q.Select().
		Fields(
			tokenLocks.Field("consumer_tx_id"), tokenLocks.Field("tx_id"), tokenLocks.Field("idx"),
			tokenRequests.Field("status"), tokenLocks.Field("created_at"), common3.FieldName("NOW() AS now"),
		).
		From(tokenLocks.Join(tokenRequests, cond.Cmp(tokenLocks.Field("consumer_tx_id"), "=", tokenRequests.Field("tx_id")))).
		Where(common.IsExpiredToken(tokenRequests, tokenLocks, leaseExpiry)).Format(db.ci)
	db.Logger.Debug(query, args)

	rows, err := db.ReadDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	it := common4.NewIterator(rows, func(entry *lockEntry) error {
		entry.LeaseExpiry = leaseExpiry
		return rows.Scan(&entry.ConsumerTxID, &entry.TokenID.TxId, &entry.TokenID.Index, &entry.Status, &entry.CreatedAt, &entry.Now)
	})
	lockEntries, err := iterators.ReadAllValues(it)
	if err != nil {
		return nil, err
	}

	db.Logger.Infof("Found following entries ready for deletion: [%v]", lockEntries)
	return lockEntries, nil
}

type lockEntry struct {
	ConsumerTxID string
	TokenID      token.ID
	Status       *driver.TxStatus
	CreatedAt    time.Time
	Now          time.Time
	LeaseExpiry  time.Duration
}

func (e lockEntry) Expired() bool {
	return e.CreatedAt.Add(e.LeaseExpiry).Before(e.Now)
}

func (e lockEntry) String() string {
	if expired := e.Expired(); e.Status == nil && expired {
		return fmt.Sprintf("Expired lock created at [%v] for token [%s] consumed by [%s]", e.CreatedAt, e.TokenID, e.ConsumerTxID)
	} else if e.Status != nil && *e.Status == driver.Deleted && !expired {
		return fmt.Sprintf("Lock created at [%v] of spent token [%s] consumed by [%s]", e.CreatedAt, e.TokenID, e.ConsumerTxID)
	} else {
		return fmt.Sprintf("Invalid token lock state: [%s] created at [%v], expired [%v], status: [%v]", e.TokenID, e.CreatedAt, expired, e.Status)
	}
}
