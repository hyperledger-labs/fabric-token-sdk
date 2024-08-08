/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
	dbdriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	sqldb "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql"
)

const (
	// OptsKey is the key for the opts in the config
	OptsKey   = "tokendb.persistence.opts"
	EnvVarKey = "TOKENDB_DATASOURCE"
)

func NewSQLDBOpener() *sqldb.DBOpener {
	return sqldb.NewSQLDBOpener(OptsKey, EnvVarKey)
}

func NewDriver() db.NamedDriver[dbdriver.TokenDBDriver] {
	return db.NamedDriver[dbdriver.TokenDBDriver]{
		Name:   sql.SQLPersistence,
		Driver: db.NewSQLDriver(NewSQLDBOpener(), sqldb.NewTokenDB),
	}
}

func NewNDBDriver() db.NamedDriver[dbdriver.TokenNDBDriver] {
	return db.NamedDriver[dbdriver.TokenNDBDriver]{
		Name:   sql.SQLPersistence,
		Driver: db.NewSQLDriver(NewSQLDBOpener(), sqldb.NewTokenNDB),
	}
}
