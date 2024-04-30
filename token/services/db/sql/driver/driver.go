/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/cache/secondcache"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditdb"
	dbdriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	sqldb "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identitydb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokendb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	"github.com/pkg/errors"
)

const (
	// optsKey is the key for the opts in the config
	optsKey   = "db.persistence.opts"
	envVarKey = "UNITYDB_DATASOURCE"
)

type Driver struct {
	DBOpener *sqldb.DBOpener
}

func NewDriver() *Driver {
	return &Driver{
		DBOpener: sqldb.NewSQLDBOpener(optsKey, envVarKey),
	}
}

func (d *Driver) OpenTokenTransactionDB(cp core.ConfigProvider, tmsID token.TMSID) (dbdriver.TokenTransactionDB, error) {
	sqlDB, opts, err := d.DBOpener.Open(cp, tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open db at [%s:%s]", optsKey, envVarKey)
	}
	return sqldb.NewTransactionDB(sqlDB, opts.TablePrefix, !opts.SkipCreateTable)
}

func (d *Driver) OpenTokenDB(cp core.ConfigProvider, tmsID token.TMSID) (dbdriver.TokenDB, error) {
	sqlDB, opts, err := d.DBOpener.Open(cp, tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open db at [%s:%s]", optsKey, envVarKey)
	}
	return sqldb.NewTokenDB(sqlDB, opts.TablePrefix, !opts.SkipCreateTable)
}

func (d *Driver) OpenAuditTransactionDB(cp core.ConfigProvider, tmsID token.TMSID) (dbdriver.AuditTransactionDB, error) {
	sqlDB, opts, err := d.DBOpener.Open(cp, tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open db at [%s:%s]", optsKey, envVarKey)
	}
	return sqldb.NewTransactionDB(sqlDB, opts.TablePrefix+"aud_", !opts.SkipCreateTable)
}

func (d *Driver) OpenWalletDB(cp core.ConfigProvider, tmsID token.TMSID) (dbdriver.WalletDB, error) {
	sqlDB, opts, err := d.DBOpener.Open(cp, tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open db at [%s:%s]", optsKey, envVarKey)
	}
	return sqldb.NewWalletDB(sqlDB, opts.TablePrefix, !opts.SkipCreateTable)
}

func (d *Driver) OpenIdentityDB(cp core.ConfigProvider, tmsID token.TMSID) (dbdriver.IdentityDB, error) {
	sqlDB, opts, err := d.DBOpener.Open(cp, tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open db at [%s:%s]", optsKey, envVarKey)
	}
	return sqldb.NewIdentityDB(sqlDB, opts.TablePrefix, !opts.SkipCreateTable, secondcache.New(1000))
}

type TtxDBDriver struct {
	*Driver
}

func (t *TtxDBDriver) Open(cp core.ConfigProvider, tmsID token.TMSID) (dbdriver.TokenTransactionDB, error) {
	return t.OpenTokenTransactionDB(cp, tmsID)
}

type TokenDBDriver struct {
	*Driver
}

func (t *TokenDBDriver) Open(cp core.ConfigProvider, tmsID token.TMSID) (dbdriver.TokenDB, error) {
	return t.OpenTokenDB(cp, tmsID)
}

type AuditDBDriver struct {
	*Driver
}

func (t *AuditDBDriver) Open(cp core.ConfigProvider, tmsID token.TMSID) (dbdriver.AuditTransactionDB, error) {
	return t.OpenAuditTransactionDB(cp, tmsID)
}

type IdentityDBDriver struct {
	*Driver
}

func (t *IdentityDBDriver) OpenWalletDB(cp core.ConfigProvider, tmsID token.TMSID) (dbdriver.WalletDB, error) {
	return t.Driver.OpenWalletDB(cp, tmsID)
}

func (t *IdentityDBDriver) OpenIdentityDB(cp core.ConfigProvider, tmsID token.TMSID) (dbdriver.IdentityDB, error) {
	return t.Driver.OpenIdentityDB(cp, tmsID)
}

func init() {
	root := NewDriver()
	ttxdb.Register("unity", &TtxDBDriver{Driver: root})
	tokendb.Register("unity", &TokenDBDriver{Driver: root})
	auditdb.Register("unity", &AuditDBDriver{Driver: root})
	identitydb.Register("unity", &IdentityDBDriver{Driver: root})
}
