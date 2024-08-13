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
)

type TokenLockDB struct {
	*common.TokenLockDB
}

func OpenTokenLockDB(k common.Opts) (driver.TokenLockDB, error) {
	db, err := postgres.OpenDB(k.DataSource, k.MaxOpenConns)
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

func (db *TokenLockDB) Cleanup(evictionDelay time.Duration) error {
	query := fmt.Sprintf(
		"DELETE FROM %s "+
			"USING %s WHERE %s.consumer_tx_id = %s.tx_id AND (%s.status IN (%d) "+
			"OR %s.created_at < NOW() - INTERVAL '%d seconds'"+
			");",
		db.Table.TokenLocks,
		db.Table.Requests, db.Table.TokenLocks, db.Table.Requests, db.Table.Requests, driver.Deleted,
		db.Table.TokenLocks, int(evictionDelay.Seconds()),
	)
	db.Logger.Debug(query)
	_, err := db.DB.Exec(query)
	if err != nil {
		db.Logger.Errorf("query failed: %s", query)
	}
	return err
}
