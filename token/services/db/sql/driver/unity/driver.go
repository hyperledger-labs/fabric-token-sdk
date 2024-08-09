/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package unity

import (
	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils"
	sql3 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql"
	common2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
	dbdriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/postgres"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/sqlite"
	"github.com/pkg/errors"
)

const (
	// optsKey is the key for the opts in the config
	optsKey   = "db.persistence.opts"
	envVarKey = "UNITYDB_DATASOURCE"

	UnityPersistence driver2.PersistenceType = "unity"
)

type constructors[D any] map[common2.SQLDriverType]common.NewDBFunc[D]

type Driver struct {
	DBOpener *common.DBOpener
}

func NewDriver() *Driver {
	return &Driver{
		DBOpener: common.NewSQLDBOpener(optsKey, envVarKey),
	}
}

func (d *Driver) OpenTokenTransactionDB(cp dbdriver.ConfigProvider, tmsID token.TMSID) (dbdriver.TokenTransactionDB, error) {
	return openDB(d.DBOpener, cp, tmsID, constructors[dbdriver.TokenTransactionDB]{
		sql3.SQLite:   sqlite.NewTransactionDB,
		sql3.Postgres: postgres.NewTransactionDB,
	})
}

func (d *Driver) OpenTokenDB(cp dbdriver.ConfigProvider, tmsID token.TMSID) (dbdriver.TokenDB, error) {
	return openDB(d.DBOpener, cp, tmsID, constructors[dbdriver.TokenDB]{
		sql3.SQLite:   sqlite.NewTokenDB,
		sql3.Postgres: postgres.NewTokenDB,
	})
}

func (d *Driver) OpenTokenLockDB(cp dbdriver.ConfigProvider, tmsID token.TMSID) (dbdriver.TokenLockDB, error) {
	return openDB(d.DBOpener, cp, tmsID, constructors[dbdriver.TokenLockDB]{
		sql3.SQLite:   sqlite.NewTokenLockDB,
		sql3.Postgres: postgres.NewTokenLockDB,
	})
}

func (d *Driver) OpenAuditTransactionDB(cp dbdriver.ConfigProvider, tmsID token.TMSID) (dbdriver.AuditTransactionDB, error) {
	return openDB(d.DBOpener, cp, tmsID, constructors[dbdriver.AuditTransactionDB]{
		sql3.SQLite:   sqlite.NewAuditTransactionDB,
		sql3.Postgres: postgres.NewAuditTransactionDB,
	})
}

func (d *Driver) OpenWalletDB(cp dbdriver.ConfigProvider, tmsID token.TMSID) (dbdriver.WalletDB, error) {
	return openDB(d.DBOpener, cp, tmsID, constructors[dbdriver.WalletDB]{
		sql3.SQLite:   sqlite.NewWalletDB,
		sql3.Postgres: postgres.NewWalletDB,
	})
}

func (d *Driver) OpenIdentityDB(cp dbdriver.ConfigProvider, tmsID token.TMSID) (dbdriver.IdentityDB, error) {
	return openDB(d.DBOpener, cp, tmsID, constructors[dbdriver.IdentityDB]{
		sql3.SQLite:   sqlite.NewIdentityDB,
		sql3.Postgres: postgres.NewIdentityDB,
	})
}

func openDB[D any](dbOpener *common.DBOpener, cp dbdriver.ConfigProvider, tmsID token.TMSID, constructors constructors[D]) (D, error) {
	sqlDB, opts, err := dbOpener.OpenWithOpts(cp, tmsID)
	if err != nil {
		return utils.Zero[D](), errors.Wrapf(err, "failed to open db at [%s:%s]", optsKey, envVarKey)
	}
	constructor, ok := constructors[opts.Driver]
	if !ok {
		return utils.Zero[D](), errors.New("constructor not found")
	}
	return constructor(sqlDB, common.NewDBOptsFromOpts(*opts))
}

type TtxDBDriver struct {
	*Driver
}

func (t *TtxDBDriver) Open(cp dbdriver.ConfigProvider, tmsID token.TMSID) (dbdriver.TokenTransactionDB, error) {
	return t.OpenTokenTransactionDB(cp, tmsID)
}

type TokenDBDriver struct {
	*Driver
}

func (t *TokenDBDriver) Open(cp dbdriver.ConfigProvider, tmsID token.TMSID) (dbdriver.TokenDB, error) {
	return t.OpenTokenDB(cp, tmsID)
}

type TokenLockDBDriver struct {
	*Driver
}

func (t *TokenLockDBDriver) Open(cp dbdriver.ConfigProvider, tmsID token.TMSID) (dbdriver.TokenLockDB, error) {
	return t.OpenTokenLockDB(cp, tmsID)
}

type AuditDBDriver struct {
	*Driver
}

func (t *AuditDBDriver) Open(cp dbdriver.ConfigProvider, tmsID token.TMSID) (dbdriver.AuditTransactionDB, error) {
	return t.OpenAuditTransactionDB(cp, tmsID)
}

type IdentityDBDriver struct {
	*Driver
}

func (t *IdentityDBDriver) OpenWalletDB(cp dbdriver.ConfigProvider, tmsID token.TMSID) (dbdriver.WalletDB, error) {
	return t.Driver.OpenWalletDB(cp, tmsID)
}

func (t *IdentityDBDriver) OpenIdentityDB(cp dbdriver.ConfigProvider, tmsID token.TMSID) (dbdriver.IdentityDB, error) {
	return t.Driver.OpenIdentityDB(cp, tmsID)
}

func NewDBDrivers() (db.NamedDriver[dbdriver.TTXDBDriver], db.NamedDriver[dbdriver.TokenDBDriver], db.NamedDriver[dbdriver.TokenLockDBDriver], db.NamedDriver[dbdriver.AuditDBDriver], db.NamedDriver[dbdriver.IdentityDBDriver]) {
	root := NewDriver()
	return db.NamedDriver[dbdriver.TTXDBDriver]{Name: UnityPersistence, Driver: &TtxDBDriver{Driver: root}},
		db.NamedDriver[dbdriver.TokenDBDriver]{Name: UnityPersistence, Driver: &TokenDBDriver{Driver: root}},
		db.NamedDriver[dbdriver.TokenLockDBDriver]{Name: UnityPersistence, Driver: &TokenLockDBDriver{Driver: root}},
		db.NamedDriver[dbdriver.AuditDBDriver]{Name: UnityPersistence, Driver: &AuditDBDriver{Driver: root}},
		db.NamedDriver[dbdriver.IdentityDBDriver]{Name: UnityPersistence, Driver: &IdentityDBDriver{Driver: root}}
}
