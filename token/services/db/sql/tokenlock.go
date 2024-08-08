/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"database/sql"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/types/transaction"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type TokenLockDB struct {
	db *sql.DB
}

func NewTokenLockDB(db *sql.DB, createSchema bool) (driver.TokenLockDB, error) {
	identityDB := &TokenLockDB{
		db: db,
	}
	if createSchema {
		if err := initSchema(db, identityDB.GetSchema()); err != nil {
			return nil, err
		}
	}
	return identityDB, nil
}

func (db *TokenLockDB) Lock(tokenID *token.ID, consumerTxID transaction.ID) error {
	query := "INSERT INTO token_locks (consumer_tx_id, tx_id, idx, created_at) VALUES ($1, $2, $3, $4)"
	logger.Debug(query, tokenID, consumerTxID)

	_, err := db.db.Exec(query, consumerTxID, tokenID.TxId, tokenID.Index, time.Now())
	return err
}

func (db *TokenLockDB) UnlockByTxID(consumerTxID transaction.ID) error {
	query := "DELETE FROM token_locks WHERE consumer_tx_id = $1"
	logger.Debug(query, consumerTxID)

	_, err := db.db.Exec(query, consumerTxID)
	return err
}

func (db *TokenLockDB) GetSchema() string {
	return `
	-- TokenLocks
	CREATE TABLE IF NOT EXISTS token_locks (
		tx_id TEXT NOT NULL,
		idx INT NOT NULL,
		consumer_tx_id TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL,
		PRIMARY KEY(tx_id, idx)
	);`
}
