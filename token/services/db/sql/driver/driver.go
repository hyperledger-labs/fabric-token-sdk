/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/cache/secondcache"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
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

func (d *Driver) OpenTokenTransactionDB(sp view.ServiceProvider, tmsID token.TMSID) (dbdriver.TokenTransactionDB, error) {
	sqlDB, opts, err := d.DBOpener.Open(sp, tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open db at [%s:%s]", optsKey, envVarKey)
	}
	return sqldb.NewTransactionDB(sqlDB, opts.TablePrefix, !opts.SkipCreateTable)
}

func (d *Driver) OpenTokenDB(sp view.ServiceProvider, tmsID token.TMSID) (dbdriver.TokenDB, error) {
	sqlDB, opts, err := d.DBOpener.Open(sp, tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open db at [%s:%s]", optsKey, envVarKey)
	}
	return sqldb.NewTokenDB(sqlDB, opts.TablePrefix, !opts.SkipCreateTable)
}

func (d *Driver) OpenAuditTransactionDB(sp view.ServiceProvider, tmsID token.TMSID) (dbdriver.AuditTransactionDB, error) {
	sqlDB, opts, err := d.DBOpener.Open(sp, tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open db at [%s:%s]", optsKey, envVarKey)
	}
	return sqldb.NewTransactionDB(sqlDB, opts.TablePrefix+"aud_", !opts.SkipCreateTable)
}

func (d *Driver) OpenWalletDB(sp view.ServiceProvider, tmsID token.TMSID) (dbdriver.WalletDB, error) {
	sqlDB, opts, err := d.DBOpener.Open(sp, tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open db at [%s:%s]", optsKey, envVarKey)
	}
	return sqldb.NewWalletDB(sqlDB, opts.TablePrefix, !opts.SkipCreateTable)
}

func (d *Driver) OpenIdentityDB(sp view.ServiceProvider, tmsID token.TMSID) (dbdriver.IdentityDB, error) {
	sqlDB, opts, err := d.DBOpener.Open(sp, tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open db at [%s:%s]", optsKey, envVarKey)
	}
	return sqldb.NewIdentityDB(sqlDB, opts.TablePrefix, !opts.SkipCreateTable, secondcache.New(1000))
}

type TtxDBDriver struct {
	*Driver
}

func (t *TtxDBDriver) Open(sp view.ServiceProvider, tmsID token.TMSID) (dbdriver.TokenTransactionDB, error) {
	return t.OpenTokenTransactionDB(sp, tmsID)
}

type TokenDBDriver struct {
	*Driver
}

func (t *TokenDBDriver) Open(sp view.ServiceProvider, tmsID token.TMSID) (dbdriver.TokenDB, error) {
	return t.OpenTokenDB(sp, tmsID)
}

type AuditDBDriver struct {
	*Driver
}

func (t *AuditDBDriver) Open(sp view.ServiceProvider, tmsID token.TMSID) (dbdriver.AuditTransactionDB, error) {
	return t.OpenAuditTransactionDB(sp, tmsID)
}

type IdentityDBDriver struct {
	*Driver
}

func (t *IdentityDBDriver) OpenWalletDB(sp view.ServiceProvider, tmsID token.TMSID) (dbdriver.WalletDB, error) {
	return t.Driver.OpenWalletDB(sp, tmsID)
}

func (t *IdentityDBDriver) OpenIdentityDB(sp view.ServiceProvider, tmsID token.TMSID) (dbdriver.IdentityDB, error) {
	return t.Driver.OpenIdentityDB(sp, tmsID)
}

func init() {
	root := NewDriver()
	ttxdb.Register("unity", &TtxDBDriver{Driver: root})
	tokendb.Register("unity", &TokenDBDriver{Driver: root})
	auditdb.Register("unity", &AuditDBDriver{Driver: root})
	identitydb.Register("unity", &IdentityDBDriver{Driver: root})
}
