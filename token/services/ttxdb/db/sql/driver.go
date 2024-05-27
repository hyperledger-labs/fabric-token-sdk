/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	sqldb "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/drivers"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
)

const (
	// OptsKey is the key for the opts in the config
	OptsKey   = "ttxdb.persistence.opts"
	EnvVarKey = "TTXDB_DATASOURCE"
)

func NewSQLDBOpener() *sqldb.DBOpener {
	return sqldb.NewSQLDBOpener(OptsKey, EnvVarKey)
}

func init() {
	ttxdb.Register("sql", drivers.NewSQLDriver(NewSQLDBOpener(), sqldb.NewTransactionDB))
}
