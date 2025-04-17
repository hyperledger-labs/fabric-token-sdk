/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"database/sql"
	"fmt"
	"time"

	common2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/common"
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
	ReadDB  *sql.DB
	WriteDB *sql.DB
	Table   tokenLockTables
	Logger  logging.Logger
}

func newTokenLockDB(readDB, writeDB *sql.DB, tables tokenLockTables) *TokenLockDB {
	return &TokenLockDB{
		ReadDB:  readDB,
		WriteDB: writeDB,
		Table:   tables,
		Logger:  logger,
	}
}

func NewTokenLockDB(readDB, writeDB *sql.DB, tables tableNames) (*TokenLockDB, error) {
	return newTokenLockDB(
		readDB,
		writeDB,
		tokenLockTables{
			TokenLocks: tables.TokenLocks,
			Requests:   tables.Requests,
		}), nil
}

func (db *TokenLockDB) CreateSchema() error {
	return common.InitSchema(db.WriteDB, []string{db.GetSchema()}...)
}

func (db *TokenLockDB) Lock(tokenID *token.ID, consumerTxID transaction.ID) error {
	query, err := NewInsertInto(db.Table.TokenLocks).Rows("consumer_tx_id, tx_id, idx, created_at").Compile()
	if err != nil {
		return errors.Wrap(err, "failed compiling query")
	}
	logger.Debug(query, tokenID, consumerTxID)
	_, err = db.WriteDB.Exec(query, consumerTxID, tokenID.TxId, tokenID.Index, time.Now().UTC())
	return err
}

func (db *TokenLockDB) UnlockByTxID(consumerTxID transaction.ID) error {
	query, err := NewDeleteFrom(db.Table.TokenLocks).Where("consumer_tx_id = $1").Compile()
	if err != nil {
		return errors.Wrap(err, "failed compiling query")
	}
	logger.Debug(query, consumerTxID)

	_, err = db.WriteDB.Exec(query, consumerTxID)
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

func (db *TokenLockDB) Close() error {
	return common2.Close(db.ReadDB, db.WriteDB)
}
