/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sqlite

import (
	"context"
	"time"

	sq "github.com/Masterminds/squirrel"
	common3 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/common"
	common4 "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/common"
)

type TokenLockStore struct {
	*common4.TokenLockStore
	pf sq.PlaceholderFormat
}

func (db *TokenLockStore) Cleanup(ctx context.Context, leaseExpiry time.Duration) error {
	tl, tr := db.Table.TokenLocks, db.Table.Requests

	subq, subArgs, err := sq.Select("tl.tx_id").
		From(tl + " AS tl").
		LeftJoin(tr + " AS tr ON tl.tx_id = tr.tx_id").
		Where(common4.IsExpiredToken("tr", "tl", leaseExpiry)).
		ToSql()
	if err != nil {
		return err
	}

	query, args, err := sq.Delete(tl).
		Where("tx_id IN ("+subq+")", subArgs...).
		PlaceholderFormat(db.pf).
		ToSql()
	if err != nil {
		return err
	}

	db.Logger.Debug(query, args)
	_, err = db.WriteDB.ExecContext(ctx, query, args...)
	if err != nil {
		db.Logger.Errorf("query failed: %s", query)
	}

	return err
}

func NewTokenLockStore(dbs *common3.RWDB, tableNames common4.TableNames) (*TokenLockStore, error) {
	tldb, err := common4.NewTokenLockStore(dbs.ReadDB, dbs.WriteDB, tableNames, sq.Question)
	if err != nil {
		return nil, err
	}

	return &TokenLockStore{TokenLockStore: tldb, pf: sq.Question}, nil
}
