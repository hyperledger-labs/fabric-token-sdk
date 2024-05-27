/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
	sqldb "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql"
)

const (
	// OptsKey is the key for the opts in the config
	OptsKey   = "auditdb.persistence.opts"
	EnvVarKey = "AUDITDB_DATASOURCE"
)

type Driver struct {
	*sqldb.DBOpener
}

func NewSQLDBOpener() *sqldb.DBOpener {
	return sqldb.NewSQLDBOpener(OptsKey, EnvVarKey)
}

func init() {
	auditdb.Register("sql", db.NewSQLDriver(NewSQLDBOpener(), sqldb.NewAuditTransactionDB))
}
