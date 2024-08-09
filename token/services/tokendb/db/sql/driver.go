/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
	dbdriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/postgres"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/sqlite"
)

const (
	// OptsKey is the key for the opts in the config
	OptsKey   = "tokendb.persistence.opts"
	EnvVarKey = "TOKENDB_DATASOURCE"
)

func NewDriver() db.NamedDriver[dbdriver.TokenDBDriver] {
	return db.NamedDriver[dbdriver.TokenDBDriver]{
		Name: sql.SQLPersistence,
		Driver: db.NewSQLDriver(func(cp dbdriver.ConfigProvider, tmsID token.TMSID) (dbdriver.TokenDB, error) {
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
		}),
	}
}

func NewNDBDriver() db.NamedDriver[dbdriver.TokenNDBDriver] {
	return db.NamedDriver[dbdriver.TokenNDBDriver]{
		Name: sql.SQLPersistence,
		Driver: db.NewSQLDriver(func(cp dbdriver.ConfigProvider, tmsID token.TMSID) (dbdriver.TokenNDB, error) {
			sqlDB, opts, err := common.NewSQLDBOpener(OptsKey, EnvVarKey).OpenWithOpts(cp, tmsID)
			if err != nil {
				return nil, err
			}
			switch opts.Driver {
			case sql.SQLite:
				return sqlite.NewTokenNDB(sqlDB, common.NewDBOptsFromOpts(*opts))
			case sql.Postgres:
				return postgres.NewTokenNDB(sqlDB, common.NewDBOptsFromOpts(*opts))
			}
			panic("undefined")
		}),
	}
}
