/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	sqldb "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identitydb"
)

const (
	// OptsKey is the key for the opts in the config
	OptsKey   = "identitydb.persistence.opts"
	EnvVarKey = "IDENTITYDB_DATASOURCE"
)

func NewSQLDBOpener() *sqldb.DBOpener {
	return sqldb.NewSQLDBOpener(OptsKey, EnvVarKey)
}

type Driver struct {
	identityDriver *db.SQLDriver[driver.IdentityDB]
	walletDriver   *db.SQLDriver[driver.WalletDB]
}

func (d *Driver) OpenIdentityDB(cp driver.ConfigProvider, tmsID token.TMSID) (driver.IdentityDB, error) {
	return d.identityDriver.Open(cp, tmsID)
}

func (d *Driver) OpenWalletDB(cp driver.ConfigProvider, tmsID token.TMSID) (driver.WalletDB, error) {
	return d.walletDriver.Open(cp, tmsID)
}

func init() {
	sqlDBOpener := NewSQLDBOpener()
	identitydb.Register("sql", &Driver{
		identityDriver: db.NewSQLDriver(sqlDBOpener, sqldb.NewCachedIdentityDB),
		walletDriver:   db.NewSQLDriver(sqlDBOpener, sqldb.NewWalletDB),
	})
}
