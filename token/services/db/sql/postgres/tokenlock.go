/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package postgres

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/postgres"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"go.uber.org/zap/zapcore"
)

type TokenLockDB struct {
	*common.TokenLockDB
}

func OpenTokenLockDB(k common.Opts) (driver.TokenLockDB, error) {
	db, err := postgres.OpenDB(k.DataSource, k.MaxOpenConns, k.MaxIdleConns, k.MaxIdleTime)
	if err != nil {
		return nil, err
	}
	return NewTokenLockDB(db, common.NewDBOptsFromOpts(k))
}

func NewTokenLockDB(db *sql.DB, k common.NewDBOpts) (driver.TokenLockDB, error) {
	tldb, err := common.NewTokenLockDB(db, k)
	if err != nil {
		return nil, err
	}
	return &TokenLockDB{TokenLockDB: tldb}, nil
}

func (db *TokenLockDB) Cleanup(leaseExpiry time.Duration) error {
	if err := db.logStaleLocks(leaseExpiry); err != nil {
		db.Logger.Warnf("Could not log stale locks: %v", err)
	}
	query := fmt.Sprintf(
		"DELETE FROM %s "+
			"USING %s WHERE %s.consumer_tx_id = %s.tx_id AND (%s.status IN (%d) "+
			"OR %s.created_at < NOW() - INTERVAL '%d seconds'"+
			");",
		db.Table.TokenLocks,
		db.Table.Requests, db.Table.TokenLocks, db.Table.Requests, db.Table.Requests, driver.Deleted,
		db.Table.TokenLocks, int(leaseExpiry.Seconds()),
	)
	db.Logger.Debug(query)
	_, err := db.DB.Exec(query)
	if err != nil {
		db.Logger.Errorf("query failed: %s", query)
	}
	return err
}

func (db *TokenLockDB) logStaleLocks(leaseExpiry time.Duration) error {
	if !db.Logger.IsEnabledFor(zapcore.InfoLevel) {
		return nil
	}
	query := fmt.Sprintf(
		"SELECT %s.consumer_tx_id, %s.tx_id, %s.idx, %s.status, %s.created_at, %s.created_at < NOW() - INTERVAL '%d seconds' AS expired "+
			"FROM %s "+
			"LEFT JOIN %s ON %s.consumer_tx_id = %s.tx_id "+
			"WHERE %s.status = %d OR %s.created_at < NOW() - INTERVAL '%d seconds'",
		db.Table.TokenLocks, db.Table.TokenLocks, db.Table.TokenLocks, db.Table.Requests, db.Table.TokenLocks, db.Table.TokenLocks, int(leaseExpiry.Seconds()),
		db.Table.TokenLocks,
		db.Table.Requests, db.Table.TokenLocks, db.Table.Requests,
		db.Table.Requests, driver.Deleted, db.Table.TokenLocks, int(leaseExpiry.Seconds()),
	)
	db.Logger.Debug(query)

	rows, err := db.DB.Query(query)
	if err != nil {
		return err
	}

	var lockEntries []lockEntry
	for rows.Next() {
		var entry lockEntry
		if err := rows.Scan(&entry.ConsumerTxID, &entry.TokenID.TxId, &entry.TokenID.Index, &entry.Status, &entry.CreatedAt, &entry.Expired); err != nil {
			return err
		}
		lockEntries = append(lockEntries, entry)
	}
	if rows.Err() != nil {
		return rows.Err()
	}
	db.Logger.Infof("Found following entries ready for deletion: [%v]", lockEntries)
	return nil
}

type lockEntry struct {
	ConsumerTxID string
	TokenID      token.ID
	Status       *driver.TxStatus
	CreatedAt    time.Time
	Expired      bool
}

func (e lockEntry) String() string {
	if e.Status == nil && e.Expired {
		return fmt.Sprintf("Expired lock created at [%v] for token [%s] consumed by [%s]", e.CreatedAt, e.TokenID, e.ConsumerTxID)
	} else if e.Status != nil && *e.Status == driver.Deleted && !e.Expired {
		return fmt.Sprintf("Lock created at [%v] of spent token [%s] consumed by [%s]", e.CreatedAt, e.TokenID, e.ConsumerTxID)
	} else {
		return fmt.Sprintf("Invalid token lock state: [%s] created at [%v], expired [%v], status: [%v]", e.TokenID, e.CreatedAt, e.Expired, e.Status)
	}
}
