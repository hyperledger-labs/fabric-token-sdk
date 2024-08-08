/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package unity

import (
	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
	dbdriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	sqldb "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql"
	"github.com/pkg/errors"
)

const (
	// optsKey is the key for the opts in the config
	optsKey   = "db.persistence.opts"
	envVarKey = "UNITYDB_DATASOURCE"

	UnityPersistence driver2.PersistenceType = "unity"
)

type Driver struct {
	DBOpener *sqldb.DBOpener
}

func NewDriver() *Driver {
	return &Driver{
		DBOpener: sqldb.NewSQLDBOpener(optsKey, envVarKey),
	}
}

func (d *Driver) OpenTokenTransactionDB(cp dbdriver.ConfigProvider, tmsID token.TMSID) (dbdriver.TokenTransactionDB, error) {
	return openDB(d.DBOpener, cp, tmsID, sqldb.NewTransactionDB)
}

func (d *Driver) OpenTokenDB(cp dbdriver.ConfigProvider, tmsID token.TMSID) (dbdriver.TokenDB, error) {
	return openDB(d.DBOpener, cp, tmsID, sqldb.NewTokenDB)
}

func (d *Driver) OpenTokenLockDB(cp dbdriver.ConfigProvider, tmsID token.TMSID) (dbdriver.TokenLockDB, error) {
	return openDB(d.DBOpener, cp, tmsID, sqldb.NewTokenLockDB)
}

func (d *Driver) OpenAuditTransactionDB(cp dbdriver.ConfigProvider, tmsID token.TMSID) (dbdriver.AuditTransactionDB, error) {
	return openDB(d.DBOpener, cp, tmsID, sqldb.NewAuditTransactionDB)
}

func (d *Driver) OpenWalletDB(cp dbdriver.ConfigProvider, tmsID token.TMSID) (dbdriver.WalletDB, error) {
	return openDB(d.DBOpener, cp, tmsID, sqldb.NewWalletDB)
}

func (d *Driver) OpenIdentityDB(cp dbdriver.ConfigProvider, tmsID token.TMSID) (dbdriver.IdentityDB, error) {
	return openDB(d.DBOpener, cp, tmsID, sqldb.NewCachedIdentityDB)
}

func openDB[D any](dbOpener *sqldb.DBOpener, cp dbdriver.ConfigProvider, tmsID token.TMSID, newDB db.NewDBFunc[D]) (D, error) {
	sqlDB, opts, err := dbOpener.Open(cp, tmsID)
	if err != nil {
		return utils.Zero[D](), errors.Wrapf(err, "failed to open db at [%s:%s]", optsKey, envVarKey)
	}
	return newDB(sqlDB, sqldb.NewDBOpts{
		DataSource:   opts.DataSource,
		TablePrefix:  opts.TablePrefix,
		CreateSchema: !opts.SkipCreateTable,
	})
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
