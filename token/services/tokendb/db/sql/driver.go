/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql"
	common2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/common"
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
		Driver: common.NewOpenerFromMap(OptsKey, EnvVarKey, map[common2.SQLDriverType]common.OpenFunc[dbdriver.TokenDB]{
			sql.SQLite:   sqlite.NewTokenDB,
			sql.Postgres: postgres.NewTokenDB,
		}),
	}
}

func NewNDBDriver() db.NamedDriver[dbdriver.TokenNDBDriver] {
	return db.NamedDriver[dbdriver.TokenNDBDriver]{
		Name: sql.SQLPersistence,
		Driver: common.NewOpenerFromMap(OptsKey, EnvVarKey, map[common2.SQLDriverType]common.OpenFunc[dbdriver.TokenNDB]{
			sql.SQLite:   sqlite.NewTokenNDB,
			sql.Postgres: postgres.NewTokenNDB,
		}),
	}
}
