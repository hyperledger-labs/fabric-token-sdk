/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
	dbdriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql"
)

const (
	// OptsKey is the key for the opts in the config
	OptsKey   = "tokenlockdb.persistence.opts"
	EnvVarKey = "TOKENLOCKDB_DATASOURCE"
)

func NewSQLDBOpener() *sql.DBOpener {
	return sql.NewSQLDBOpener(OptsKey, EnvVarKey)
}

func NewDriver() db.NamedDriver[dbdriver.TokenLockDBDriver] {
	return db.NamedDriver[dbdriver.TokenLockDBDriver]{
		Name:   "sql",
		Driver: db.NewMemoryDriver(NewSQLDBOpener(), sql.NewTokenLockDB),
	}
}
