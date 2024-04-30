/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core"
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

func (d *Driver) Open(cp core.ConfigProvider, tmsID token.TMSID) (driver.TokenDB, error) {
	sqlDB, opts, err := d.DBOpener.Open(cp, tmsID)
	if err != nil {
		return nil, err
	}
	return sqldb.NewTokenDB(sqlDB, opts.TablePrefix, !opts.SkipCreateTable)
}

func init() {
	tokendb.Register("sql", NewDriver())
}
