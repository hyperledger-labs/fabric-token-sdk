/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sqlite

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
)

type TokenLockDB struct {
	*common.TokenLockDB
}

func (db *TokenLockDB) Cleanup(leaseExpiry time.Duration) error {
	query := fmt.Sprintf(
		"DELETE FROM %s WHERE tx_id IN ("+
			"SELECT %s.tx_id FROM %s JOIN %s ON %s.tx_id = %s.tx_id WHERE %s.status IN (%d) "+
			"OR %s.created_at < datetime('now', '-%d seconds')"+
			");",
		db.Table.TokenLocks,
		db.Table.TokenLocks, db.Table.TokenLocks, db.Table.Requests, db.Table.TokenLocks, db.Table.Requests, db.Table.Requests, driver.Deleted,
		db.Table.TokenLocks, int(leaseExpiry.Seconds()),
	)
	db.Logger.Debug(query)
	_, err := db.WriteDB.Exec(query)
	if err != nil {
		db.Logger.Errorf("query failed: %s", query)
	}
	return err
}

func NewTokenLockDB(readDB, writeDB *sql.DB, k common.NewDBOpts) (driver.TokenLockDB, error) {
	tldb, err := common.NewTokenLockDB(readDB, writeDB, k)
	if err != nil {
		return nil, err
	}
	return &TokenLockDB{TokenLockDB: tldb}, nil
}
