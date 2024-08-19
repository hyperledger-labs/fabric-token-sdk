/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/types/transaction"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type tokenLockTables struct {
	TokenLocks string
	Requests   string
}

type TokenLockDB struct {
	DB     *sql.DB
	Table  tokenLockTables
	Logger logging.Logger
}

func newTokenLockDB(db *sql.DB, tables tokenLockTables) *TokenLockDB {
	return &TokenLockDB{
		DB:     db,
		Table:  tables,
		Logger: logger,
	}
}

func NewTokenLockDB(db *sql.DB, opts NewDBOpts) (*TokenLockDB, error) {
	tables, err := GetTableNames(opts.TablePrefix)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get table names")
	}

	tokenLockDB := newTokenLockDB(db, tokenLockTables{TokenLocks: tables.TokenLocks, Requests: tables.Requests})
	if opts.CreateSchema {
		if err = common.InitSchema(db, []string{tokenLockDB.GetSchema()}...); err != nil {
			return nil, err
		}
	}
	return tokenLockDB, nil
}

func (db *TokenLockDB) Lock(tokenID *token.ID, consumerTxID transaction.ID) error {
	query := fmt.Sprintf("INSERT INTO %s (consumer_tx_id, tx_id, idx, created_at) VALUES ($1, $2, $3, $4)", db.Table.TokenLocks)
	logger.Debug(query, tokenID, consumerTxID)

	_, err := db.DB.Exec(query, consumerTxID, tokenID.TxId, tokenID.Index, time.Now().UTC())
	return err
}

func (db *TokenLockDB) UnlockByTxID(consumerTxID transaction.ID) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE consumer_tx_id = $1", db.Table.TokenLocks)
	logger.Debug(query, consumerTxID)

	_, err := db.DB.Exec(query, consumerTxID)
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
		db.Table.TokenLocks,
	)
}
