/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/drivers"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokenlockdb"
)

const (
	// OptsKey is the key for the opts in the config
	OptsKey   = "tokenlockdb.persistence.opts"
	EnvVarKey = "TOKENLOCKDB_DATASOURCE"
)

func NewSQLDBOpener() *sql.DBOpener {
	return sql.NewSQLDBOpener(OptsKey, EnvVarKey)
}

func init() {
	tokenlockdb.Register("sql", drivers.NewSQLDriver(NewSQLDBOpener(), sql.NewTokenLockDB))
}
