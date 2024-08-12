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

func NewDBDriver() db.NamedDriver[dbdriver.TokenDBDriver] {
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

func NewNotifierDriver() db.NamedDriver[dbdriver.TokenNotifierDriver] {
	return db.NamedDriver[dbdriver.TokenNotifierDriver]{
		Name: mem.MemoryPersistence,
		Driver: db.NewSQLDriver(func(cp dbdriver.ConfigProvider, tmsID token.TMSID) (dbdriver.TokenNotifier, error) {
			sqlDB, opts, err := common.NewSQLDBOpener(sql.OptsKey, sql.EnvVarKey).OpenWithOpts(cp, tmsID)
			if err != nil {
				return nil, err
			}
			return sqlite.NewTokenNotifier(sqlDB, common.NewDBOptsFromOpts(*opts))
		}),
	}
}
