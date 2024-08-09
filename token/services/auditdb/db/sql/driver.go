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
	OptsKey   = "auditdb.persistence.opts"
	EnvVarKey = "AUDITDB_DATASOURCE"
)

type Driver struct {
	*common.DBOpener
}

func NewDriver() db.NamedDriver[dbdriver.AuditDBDriver] {
	return db.NamedDriver[dbdriver.AuditDBDriver]{
		Name: sql.SQLPersistence,
		Driver: common.NewOpenerFromMap(OptsKey, EnvVarKey, map[common2.SQLDriverType]common.OpenDBFunc[dbdriver.AuditTransactionDB]{
			sql.SQLite:   sqlite.OpenAuditTransactionDB,
			sql.Postgres: postgres.OpenAuditTransactionDB,
		}),
	}
}
