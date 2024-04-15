/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	sqldb "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokendb"
)

const (
	// OptsKey is the key for the opts in the config
	OptsKey   = "tokendb.persistence.opts"
	EnvVarKey = "TOKENDB_DATASOURCE"
)

type Driver struct {
	*sqldb.DBOpener
}

func NewDriver() *Driver {
	return &Driver{DBOpener: sqldb.NewSQLDBOpener(OptsKey, EnvVarKey)}
}

func (d *Driver) Open(sp view.ServiceProvider, tmsID token.TMSID) (driver.TokenDB, error) {
	sqlDB, opts, err := d.DBOpener.Open(sp, tmsID)
	if err != nil {
		return nil, err
	}
	return sqldb.NewTokenDB(sqlDB, opts.TablePrefix, opts.CreateSchema)
}

func init() {
	tokendb.Register("sql", NewDriver())
}
