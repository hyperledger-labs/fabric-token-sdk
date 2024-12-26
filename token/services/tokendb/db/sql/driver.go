/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	dbdriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	sql2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/postgres"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/sqlite"
)

const (
	// OptsKey is the key for the opts in the config
	OptsKey   = "tokendb.persistence.opts"
	EnvVarKey = "TOKENDB_DATASOURCE"
)

func NewDBDriver() *sql2.Driver[dbdriver.TokenDB] {
	return sql2.NewDriver(func(cp dbdriver.ConfigProvider, tmsID token.TMSID) (dbdriver.TokenDB, error) {
		sqlDB, opts, err := common.NewSQLDBOpener(OptsKey, EnvVarKey).OpenWithOpts(cp, tmsID)
		if err != nil {
			return nil, err
		}
		switch opts.Driver {
		case sql.SQLite:
			return sqlite.NewTokenDB(sqlDB, common.NewDBOptsFromOpts(*opts))
		case sql.Postgres:
			return postgres.NewTokenDB(sqlDB, common.NewDBOptsFromOpts(*opts))
		}
		panic("undefined")
	})
}

func NewNotifierDriver() dbdriver.TokenNotifierDriver {
	return sql2.NewDriver(func(cp dbdriver.ConfigProvider, tmsID token.TMSID) (dbdriver.TokenNotifier, error) {
		sqlDB, opts, err := common.NewSQLDBOpener(OptsKey, EnvVarKey).OpenWithOpts(cp, tmsID)
		if err != nil {
			return nil, err
		}
		switch opts.Driver {
		case sql.SQLite:
			return sqlite.NewTokenNotifier(sqlDB, common.NewDBOptsFromOpts(*opts))
		case sql.Postgres:
			// Make sure the schema for the table is created
			if _, err := postgres.NewTokenDB(sqlDB, common.NewDBOptsFromOpts(*opts)); err != nil {
				panic(err)
			}
			return postgres.NewTokenNotifier(sqlDB, common.NewDBOptsFromOpts(*opts))
		}
		panic("undefined")
	})
}
