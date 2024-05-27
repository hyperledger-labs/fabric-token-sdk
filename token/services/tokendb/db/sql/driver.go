/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
	sqldb "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokendb"
)

const (
	// OptsKey is the key for the opts in the config
	OptsKey   = "tokendb.persistence.opts"
	EnvVarKey = "TOKENDB_DATASOURCE"
)

func NewSQLDBOpener() *sqldb.DBOpener {
	return sqldb.NewSQLDBOpener(OptsKey, EnvVarKey)
}

func init() {
	tokendb.Register("sql", db.NewSQLDriver(NewSQLDBOpener(), sqldb.NewTokenDB))
}
