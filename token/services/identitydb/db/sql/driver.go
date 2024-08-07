/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql"
	common2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/postgres"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/sqlite"
)

const (
	// OptsKey is the key for the opts in the config
	OptsKey   = "identitydb.persistence.opts"
	EnvVarKey = "IDENTITYDB_DATASOURCE"
)

type Driver struct {
	identityDriver *common.Opener[driver.IdentityDB]
	walletDriver   *common.Opener[driver.WalletDB]
}

func (d *Driver) OpenIdentityDB(cp driver.ConfigProvider, tmsID token.TMSID) (driver.IdentityDB, error) {
	return d.identityDriver.Open(cp, tmsID)
}

func (d *Driver) OpenWalletDB(cp driver.ConfigProvider, tmsID token.TMSID) (driver.WalletDB, error) {
	return d.walletDriver.Open(cp, tmsID)
}

func NewDriver() db.NamedDriver[driver.IdentityDBDriver] {
	return db.NamedDriver[driver.IdentityDBDriver]{
		Name: sql.SQLPersistence,
		Driver: &Driver{
			identityDriver: common.NewOpenerFromMap(OptsKey, EnvVarKey, map[common2.SQLDriverType]common.OpenFunc[driver.IdentityDB]{
				sql.SQLite:   sqlite.NewCachedIdentityDB,
				sql.Postgres: postgres.NewCachedIdentityDB,
			}),
			walletDriver: common.NewOpenerFromMap(OptsKey, EnvVarKey, map[common2.SQLDriverType]common.OpenFunc[driver.WalletDB]{
				sql.SQLite:   sqlite.NewWalletDB,
				sql.Postgres: postgres.NewWalletDB,
			}),
		},
	}
}
