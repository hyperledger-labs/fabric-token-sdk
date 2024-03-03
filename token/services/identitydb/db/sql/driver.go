/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/cache/secondcache"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	sqldb "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identitydb"
)

const (
	// OptsKey is the key for the opts in the config
	OptsKey   = "identitydb.persistence.opts"
	EnvVarKey = "IDENTITYDB_DATASOURCE"
)

type Driver struct {
	*sqldb.DBOpener
}

func NewDriver() *Driver {
	return &Driver{DBOpener: sqldb.NewSQLDBOpener(OptsKey, EnvVarKey)}
}

func (d *Driver) OpenIdentityDB(sp view.ServiceProvider, tmsID token.TMSID) (driver.IdentityDB, error) {
	sqlDB, opts, err := d.DBOpener.Open(sp, tmsID)
	if err != nil {
		return nil, err
	}
	return sqldb.NewIdentityDB(sqlDB, opts.TablePrefix, opts.CreateSchema, secondcache.New(1000))
}

func (d *Driver) OpenWalletDB(sp view.ServiceProvider, tmsID token.TMSID) (driver.WalletDB, error) {
	sqlDB, opts, err := d.DBOpener.Open(sp, tmsID)
	if err != nil {
		return nil, err
	}
	return sqldb.NewWalletDB(sqlDB, opts.TablePrefix, opts.CreateSchema)
}

func init() {
	identitydb.Register("sql", NewDriver())
}
