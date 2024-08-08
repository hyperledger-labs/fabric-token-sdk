/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package memory

import (
	mem "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/memory"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
	dbdriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/sqlite"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokendb/db/sql"
)

func NewDriver() db.NamedDriver[dbdriver.TokenDBDriver] {
	return db.NamedDriver[dbdriver.TokenDBDriver]{
		Name: mem.MemoryPersistence,
		Driver: db.NewSQLDriver(func(cp dbdriver.ConfigProvider, tmsID token.TMSID) (dbdriver.TokenDB, error) {
			sqlDB, opts, err := common.NewSQLDBOpener(sql.OptsKey, sql.EnvVarKey).OpenWithOpts(cp, tmsID)
			if err != nil {
				return nil, err
			}
			return sqlite.NewTokenDB(sqlDB, common.NewDBOptsFromOpts(*opts))
		}),
	}
}

func NewNDBDriver() db.NamedDriver[dbdriver.TokenNDBDriver] {
	return db.NamedDriver[dbdriver.TokenNDBDriver]{
		Name: mem.MemoryPersistence,
		Driver: db.NewSQLDriver(func(cp dbdriver.ConfigProvider, tmsID token.TMSID) (dbdriver.TokenNDB, error) {
			sqlDB, opts, err := common.NewSQLDBOpener(sql.OptsKey, sql.EnvVarKey).OpenWithOpts(cp, tmsID)
			if err != nil {
				return nil, err
			}
			return sqlite.NewTokenNDB(sqlDB, common.NewDBOptsFromOpts(*opts))
		}),
	}
}
