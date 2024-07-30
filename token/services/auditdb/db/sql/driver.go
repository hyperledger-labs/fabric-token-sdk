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
	OptsKey   = "auditdb.persistence.opts"
	EnvVarKey = "AUDITDB_DATASOURCE"
)

type Driver struct {
	*sqldb.DBOpener
}

func NewSQLDBOpener() *sqldb.DBOpener {
	return sqldb.NewSQLDBOpener(OptsKey, EnvVarKey)
}

func NewDriver() db.NamedDriver[dbdriver.AuditDBDriver] {
	return db.NamedDriver[dbdriver.AuditDBDriver]{
		Name:   sql.SQLPersistence,
		Driver: db.NewSQLDriver(NewSQLDBOpener(), sqldb.NewAuditTransactionDB),
	}
}
