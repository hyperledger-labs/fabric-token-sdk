/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	sqldb "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql"
)

const (
	// OptsKey is the key for the opts in the config
	OptsKey   = "ttxdb.persistence.opts"
	EnvVarKey = "TTXDB_DATASOURCE"
)

func NewSQLDBOpener() *sqldb.DBOpener {
	return sqldb.NewSQLDBOpener(OptsKey, EnvVarKey)
}

func NewDriver() db.NamedDriver[driver.TTXDBDriver] {
	return db.NamedDriver[driver.TTXDBDriver]{
		Name:   "sql",
		Driver: db.NewSQLDriver(NewSQLDBOpener(), sqldb.NewTransactionDB),
	}
}
