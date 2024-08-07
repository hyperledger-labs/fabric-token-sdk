/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	sql2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
	dbdriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	common2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/postgres"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/sqlite"
)

const (
	// OptsKey is the key for the opts in the config
	OptsKey   = "tokenlockdb.persistence.opts"
	EnvVarKey = "TOKENLOCKDB_DATASOURCE"
)

func NewDriver() db.NamedDriver[dbdriver.TokenLockDBDriver] {
	return db.NamedDriver[dbdriver.TokenLockDBDriver]{
		Name: sql2.SQLPersistence,
		Driver: common2.NewOpenerFromMap(OptsKey, EnvVarKey, map[common.SQLDriverType]common2.OpenFunc[dbdriver.TokenLockDB]{
			sql2.SQLite:   sqlite.NewTokenLockDB,
			sql2.Postgres: postgres.NewTokenLockDB,
		}),
	}
}
