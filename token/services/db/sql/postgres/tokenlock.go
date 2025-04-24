/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package postgres

import (
	"fmt"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/postgres"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"go.uber.org/zap/zapcore"
)

type TokenLockStore struct {
	*common.TokenLockStore
}

func NewTokenLockStore(opts postgres.Opts) (*TokenLockStore, error) {
	dbs, err := postgres.DbProvider.OpenDB(opts)
	if err != nil {
		return nil, err
	}
	tableNames, err := common.GetTableNames(opts.TablePrefix, opts.TableNameParams...)
	if err != nil {
		return nil, err
	}
	tldb, err := common.NewTokenLockStore(dbs.ReadDB, dbs.WriteDB, tableNames)
	if err != nil {
		return nil, err
	}
	return &TokenLockStore{TokenLockStore: tldb}, nil
}

func (db *TokenLockStore) Cleanup(leaseExpiry time.Duration) error {
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
	_, err := db.WriteDB.Exec(query)
	if err != nil {
		db.Logger.Errorf("query failed: %s", query)
	}
	return err
}

func (db *TokenLockStore) logStaleLocks(leaseExpiry time.Duration) error {
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

	rows, err := db.ReadDB.Query(query)
	if err != nil {
		return err
	}
	defer common.Close(rows)

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
