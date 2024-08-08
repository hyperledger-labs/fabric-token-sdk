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

type SchemaProvider interface {
	GetSchema() string
}

func NewTokenLockDB(db *sql.DB, sqlDriver string, tablePrefix string, createSchema bool) (driver.TokenLockDB, error) {
	tables, err := getTableNames(tablePrefix)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get table names")
	}

	baseTokenLockDB := newTokenLockDB(
		db,
		tokenLockTables{
			TokenLocks: tables.TokenLocks,
			Requests:   tables.Requests,
		},
	)
	var tokenLockDB driver.TokenLockDB
	var schemaProvider SchemaProvider

	switch sqlDriver {
	case "sqlite":
		db := NewSQLiteTokenLockDB(baseTokenLockDB)
		schemaProvider = db
		tokenLockDB = db
	case "pgx":
		db := NewPostgresTokenLockDB(baseTokenLockDB)
		schemaProvider = db
		tokenLockDB = db
	default:
		logger.Warn("sql driver [%s] not recognized, use default implementation", sqlDriver)
		schemaProvider = baseTokenLockDB
		tokenLockDB = baseTokenLockDB
	}

	if createSchema {
		if err = initSchema(db, schemaProvider.GetSchema()); err != nil {
			return nil, err
		}
	}
	return tokenLockDB, nil
}

type tokenLockTables struct {
	TokenLocks string
	Requests   string
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

func (db *TokenLockDB) Lock(tokenID *token.ID, consumerTxID transaction.ID) error {
	query := fmt.Sprintf("INSERT INTO %s (consumer_tx_id, tx_id, idx, created_at) VALUES ($1, $2, $3, $4)", db.table.TokenLocks)
	logger.Debug(query, tokenID, consumerTxID)

	_, err := db.db.Exec(query, consumerTxID, tokenID.TxId, tokenID.Index, time.Now().UTC())
	return err
}

func (db *TokenLockDB) UnlockByTxID(consumerTxID transaction.ID) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE consumer_tx_id = $1", db.table.TokenLocks)
	logger.Debug(query, consumerTxID)

	_, err := db.db.Exec(query, consumerTxID)
	return err
}

// Cleanup removes the locks such that either:
// 1. The transaction that locked that token is valid or invalid;
// 2. The lock is too old.
func (db *TokenLockDB) Cleanup(evictionDelay time.Duration) error {
	query := fmt.Sprintf(
		"DELETE FROM %s JOIN %s ON %s.consumer_tx_id = %s.tx_id WHERE %s.status == 3",
		db.table.TokenLocks,
		db.table.Requests, db.table.TokenLocks, db.table.Requests,
		db.table.Requests,
	)
	logger.Debug(query)
	_, err := db.db.Exec(query)
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

type SQLiteTokenLockDB struct {
	*TokenLockDB
}

func NewSQLiteTokenLockDB(tokenLockDB *TokenLockDB) *SQLiteTokenLockDB {
	return &SQLiteTokenLockDB{TokenLockDB: tokenLockDB}
}

func (db *SQLiteTokenLockDB) Cleanup(evictionDelay time.Duration) error {
	query := fmt.Sprintf(
		"DELETE FROM %s WHERE tx_id IN ("+
			"SELECT %s.tx_id FROM %s JOIN %s ON %s.tx_id = %s.tx_id WHERE %s.status IN (3) "+
			"OR %s.created_at < datetime('now', '-%d seconds')"+
			");",
		db.table.TokenLocks,
		db.table.TokenLocks, db.table.TokenLocks, db.table.Requests, db.table.TokenLocks, db.table.Requests, db.table.Requests,
		db.table.TokenLocks, int(evictionDelay.Seconds()),
	)
	logger.Debug(query)
	_, err := db.db.Exec(query)
	return err
}

type PostgresTokenLockDB struct {
	*TokenLockDB
}

func NewPostgresTokenLockDB(tokenLockDB *TokenLockDB) *PostgresTokenLockDB {
	return &PostgresTokenLockDB{TokenLockDB: tokenLockDB}
}

func (db *PostgresTokenLockDB) Cleanup(evictionDelay time.Duration) error {
	query := fmt.Sprintf(
		"DELETE FROM %s "+
			"USING %s WHERE %s.consumer_tx_id = %s.tx_id AND (%s.status IN (3) "+
			"OR %s.created_at < NOW() - INTERVAL '%d seconds'"+
			");",
		db.table.TokenLocks,
		db.table.Requests, db.table.TokenLocks, db.table.Requests, db.table.Requests,
		db.table.TokenLocks, int(evictionDelay.Seconds()),
	)
	logger.Debug(query)
	_, err := db.db.Exec(query)
	return err
}
