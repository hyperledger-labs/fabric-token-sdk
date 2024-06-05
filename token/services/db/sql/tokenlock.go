/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/types/transaction"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type tokenLockTables struct {
	TokenLocks string
}

type TokenLockDB struct {
	db    *sql.DB
	table tokenLockTables
}

func newTokenLockDB(db *sql.DB, tables tokenLockTables) *TokenLockDB {
	return &TokenLockDB{
		db:    db,
		table: tables,
	}
}

func NewTokenLockDB(db *sql.DB, tablePrefix string, createSchema bool) (driver.TokenLockDB, error) {
	tables, err := getTableNames(tablePrefix)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get table names")
	}

	identityDB := newTokenLockDB(db, tokenLockTables{TokenLocks: tables.TokenLocks})
	if createSchema {
		if err = initSchema(db, identityDB.GetSchema()); err != nil {
			return nil, err
		}
	}
	return identityDB, nil
}

func (db *TokenLockDB) Lock(tokenID *token.ID, consumerTxID transaction.ID) error {
	query := fmt.Sprintf("INSERT INTO %s (consumer_tx_id, tx_id, idx, created_at) VALUES ($1, $2, $3, $4)", db.table.TokenLocks)
	logger.Debug(query, tokenID, consumerTxID)

	_, err := db.db.Exec(query, consumerTxID, tokenID.TxId, tokenID.Index, time.Now())
	return err
}

func (db *TokenLockDB) UnlockByTxID(consumerTxID transaction.ID) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE consumer_tx_id = $1", db.table.TokenLocks)
	logger.Debug(query, consumerTxID)

	_, err := db.db.Exec(query, consumerTxID)
	return err
}

func (db *TokenLockDB) GetSchema() string {
	return fmt.Sprintf(`
		-- TokenLocks
		CREATE TABLE IF NOT EXISTS %s (
			tx_id TEXT NOT NULL,
			idx INT NOT NULL,
			consumer_tx_id TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL,
			PRIMARY KEY(tx_id, idx)
		);`,
		db.table.TokenLocks,
	)
}
