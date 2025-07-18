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

	common2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/common"
	q "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/query"
	common3 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/query/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/query/cond"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/utils/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/utils/types/transaction"
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
	ci      common3.CondInterpreter
}

func newTokenLockStore(readDB, writeDB *sql.DB, tables tokenLockTables, ci common3.CondInterpreter) *TokenLockStore {
	return &TokenLockStore{
		ReadDB:  readDB,
		WriteDB: writeDB,
		Table:   tables,
		Logger:  logger,
		ci:      ci,
	}
}

func NewTokenLockStore(readDB, writeDB *sql.DB, tables TableNames, ci common3.CondInterpreter) (*TokenLockStore, error) {
	return newTokenLockStore(
		readDB,
		writeDB,
		tokenLockTables{
			TokenLocks: tables.TokenLocks,
			Requests:   tables.Requests,
		},
		ci), nil
}

func (db *TokenLockStore) CreateSchema() error {
	return common.InitSchema(db.WriteDB, []string{db.GetSchema()}...)
}

func (db *TokenLockStore) Lock(ctx context.Context, tokenID *token.ID, consumerTxID transaction.ID) error {
	query, args := q.InsertInto(db.Table.TokenLocks).
		Fields("consumer_tx_id", "tx_id", "idx", "created_at").
		Row(consumerTxID, tokenID.TxId, tokenID.Index, time.Now().UTC()).
		Format()
	logger.Debug(query, tokenID, consumerTxID)
	_, err := db.WriteDB.ExecContext(ctx, query, args...)
	return err
}

func (db *TokenLockStore) UnlockByTxID(ctx context.Context, consumerTxID transaction.ID) error {
	query, args := q.DeleteFrom(db.Table.TokenLocks).
		Where(cond.Eq("consumer_tx_id", consumerTxID)).
		Format(db.ci)
	logger.Debug(query, consumerTxID)

	_, err := db.WriteDB.ExecContext(ctx, query, args...)
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

func IsExpiredToken(tokenRequests, tokenLocks common3.Table, leaseExpiry time.Duration) cond.Condition {
	return cond.Or(
		cond.FieldIn(tokenRequests.Field("status"), driver.Deleted),
		cond.OlderThan(tokenLocks.Field("created_at"), leaseExpiry),
	)
}
