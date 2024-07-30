/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package db

import (
	"database/sql"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	sqldb "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql"
)

type sqlDBOpener interface {
	Open(cp ConfigProvider, tmsID token.TMSID) (*sql.DB, *sqldb.Opts, error)
}

type SQLDriver[D any] struct {
	sqlDBOpener sqlDBOpener
	newDB       NewDBFunc[D]
}

func NewSQLDriver[D any](sqlDBOpener sqlDBOpener, newDB NewDBFunc[D]) *SQLDriver[D] {
	return &SQLDriver[D]{sqlDBOpener: sqlDBOpener, newDB: newDB}
}

func (d *SQLDriver[D]) Open(cp ConfigProvider, tmsID token.TMSID) (D, error) {
	sqlDB, opts, err := d.sqlDBOpener.Open(cp, tmsID)
	if err != nil {
		return utils.Zero[D](), err
	}
	return d.newDB(sqlDB, sqldb.NewDBOpts{
		DataSource:   opts.DataSource,
		TablePrefix:  opts.TablePrefix,
		CreateSchema: !opts.SkipCreateTable,
	})
}
