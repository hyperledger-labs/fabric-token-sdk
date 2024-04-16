/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
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

func NewDriver() *Driver {
	return &Driver{DBOpener: sqldb.NewSQLDBOpener(OptsKey, EnvVarKey)}
}

func (d *Driver) Open(sp view.ServiceProvider, tmsID token.TMSID) (driver.AuditTransactionDB, error) {
	sqlDB, opts, err := d.DBOpener.Open(sp, tmsID)
	if err != nil {
		return nil, err
	}
	return sqldb.NewTransactionDB(sqlDB, opts.TablePrefix+"_aud", !opts.SkipCreateTable)
}

func init() {
	auditdb.Register("sql", NewDriver())
}
