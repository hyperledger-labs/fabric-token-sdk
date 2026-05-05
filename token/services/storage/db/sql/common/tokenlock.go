/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	common2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/types/transaction"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type tokenLockTables struct {
	TokenLocks string
	Requests   string
}

type TokenLockStore struct {
	ReadDB  *sql.DB
	WriteDB *sql.DB
	Table   tokenLockTables
	Logger  logging.Logger
	pf      sq.PlaceholderFormat
}

func newTokenLockStore(readDB, writeDB *sql.DB, tables tokenLockTables, pf sq.PlaceholderFormat) *TokenLockStore {
	return &TokenLockStore{
		ReadDB:  readDB,
		WriteDB: writeDB,
		Table:   tables,
		Logger:  logger,
		pf:      pf,
	}
}

func NewTokenLockStore(readDB, writeDB *sql.DB, tables TableNames, pf sq.PlaceholderFormat) (*TokenLockStore, error) {
	return newTokenLockStore(
		readDB,
		writeDB,
		tokenLockTables{
			TokenLocks: tables.TokenLocks,
			Requests:   tables.Requests,
		},
		pf), nil
}

func (db *TokenLockStore) CreateSchema() error {
	return common.InitSchema(db.WriteDB, []string{db.GetSchema()}...)
}

func (db *TokenLockStore) Lock(ctx context.Context, tokenID *token.ID, consumerTxID transaction.ID) error {
	query, args, err := sq.Insert(db.Table.TokenLocks).
		Columns("consumer_tx_id", "tx_id", "idx", "created_at").
		Values(consumerTxID, tokenID.TxId, tokenID.Index, time.Now().UTC()).
		PlaceholderFormat(db.pf).
		ToSql()
	if err != nil {
		return err
	}
	logging.Debug(logger, query, tokenID, consumerTxID)
	_, err = db.WriteDB.ExecContext(ctx, query, args...)

	return err
}

func (db *TokenLockStore) UnlockByTxID(ctx context.Context, consumerTxID transaction.ID) error {
	query, args, err := sq.Delete(db.Table.TokenLocks).
		Where(sq.Eq{"consumer_tx_id": consumerTxID}).
		PlaceholderFormat(db.pf).
		ToSql()
	if err != nil {
		return err
	}
	logging.Debug(logger, query, consumerTxID)
	_, err = db.WriteDB.ExecContext(ctx, query, args...)

	return err
}

func (db *TokenLockStore) GetSchema() string {
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

func (db *TokenLockStore) Close() error {
	return common2.Close(db.ReadDB, db.WriteDB)
}

// IsExpiredToken returns a squirrel condition that matches token locks that are either
// deleted (status = Deleted in the requests table) or older than the given leaseExpiry.
// tokenRequestsAlias and tokenLocksAlias are the SQL aliases used in the query.
func IsExpiredToken(tokenRequestsAlias, tokenLocksAlias string, leaseExpiry time.Duration) sq.Sqlizer {
	return sq.Or{
		sq.Eq{tokenRequestsAlias + ".status": driver.Deleted},
		sq.Lt{tokenLocksAlias + ".created_at": time.Now().UTC().Add(-leaseExpiry)},
	}
}
