/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"database/sql"
	"fmt"
	"os"
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/cache/secondcache"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditdb"
	auditdbd "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	sql2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identitydb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokendb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("token-sdk.sql")

const (
	// OptsKey is the key for the opts in the config
	OptsKey   = "db.persistence.opts"
	EnvVarKey = "UNITYDB_DATASOURCE"
)

type Opts struct {
	Driver       string
	DataSource   string
	CreateSchema bool
	MaxOpenConns int
	TablePrefix  string
}

type Driver struct {
	mutex sync.RWMutex
	dbs   map[string]*sql.DB
}

func NewDriver() *Driver {
	return &Driver{dbs: make(map[string]*sql.DB)}
}

func (d *Driver) OpenTokenTransactionDB(sp view.ServiceProvider, tmsID token.TMSID) (auditdbd.TokenTransactionDB, error) {
	sqlDB, opts, err := d.open(sp, tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open db at [%s:%s:%s]", OptsKey, EnvVarKey, opts.Driver)
	}
	return sql2.NewTransactionDB(sqlDB, opts.TablePrefix, opts.CreateSchema)
}

func (d *Driver) OpenTokenDB(sp view.ServiceProvider, tmsID token.TMSID) (auditdbd.TokenDB, error) {
	sqlDB, opts, err := d.open(sp, tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open db at [%s:%s:%s]", OptsKey, EnvVarKey, opts.Driver)
	}
	return sql2.NewTokenDB(sqlDB, opts.TablePrefix, opts.CreateSchema)
}

func (d *Driver) OpenAuditTransactionDB(sp view.ServiceProvider, tmsID token.TMSID) (auditdbd.AuditTransactionDB, error) {
	sqlDB, opts, err := d.open(sp, tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open db at [%s:%s:%s]", OptsKey, EnvVarKey, opts.Driver)
	}
	return sql2.NewTransactionDB(sqlDB, opts.TablePrefix+"aud_", opts.CreateSchema)
}

func (d *Driver) OpenWalletDB(sp view.ServiceProvider, tmsID token.TMSID) (auditdbd.WalletDB, error) {
	sqlDB, opts, err := d.open(sp, tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open db at [%s:%s:%s]", OptsKey, EnvVarKey, opts.Driver)
	}
	return sql2.NewWalletDB(sqlDB, opts.TablePrefix, opts.CreateSchema)
}

func (d *Driver) OpenIdentityDB(sp view.ServiceProvider, tmsID token.TMSID) (auditdbd.IdentityDB, error) {
	sqlDB, opts, err := d.open(sp, tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open db at [%s:%s:%s]", OptsKey, EnvVarKey, opts.Driver)
	}
	return sql2.NewIdentityDB(sqlDB, opts.TablePrefix, opts.CreateSchema, secondcache.New(1000))
}

func (d *Driver) open(sp view.ServiceProvider, tmsID token.TMSID) (*sql.DB, *Opts, error) {
	opts := &Opts{}
	tmsConfig, err := config.NewTokenSDK(view.GetConfigService(sp)).GetTMS(tmsID.Network, tmsID.Channel, tmsID.Namespace)
	if err != nil {
		return nil, nil, errors.WithMessagef(err, "failed to load configuration for tms [%s]", tmsID)
	}
	if err := tmsConfig.UnmarshalKey(OptsKey, opts); err != nil {
		return nil, nil, errors.Wrapf(err, "failed getting opts for vault")
	}
	if opts.Driver == "" {
		panic(fmt.Sprintf("%s.driver not set", OptsKey))
	}

	dataSourceName := os.Getenv(EnvVarKey)
	if dataSourceName == "" {
		dataSourceName = opts.DataSource
	}
	if dataSourceName == "" {
		return nil, nil, errors.Errorf("either %s.dataSource in core.yaml or %s"+
			"environment variable must be set to a dataSourceName that can be used with the %s golang driver",
			OptsKey, EnvVarKey, opts.Driver)
	}

	sqlDB, err := d.openSQLDB(opts.Driver, opts.DataSource, opts.MaxOpenConns)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to open db at [%s:%s:%s]", OptsKey, EnvVarKey, opts.Driver)
	}

	return sqlDB, opts, nil
}

func (d *Driver) openSQLDB(driverName, dataSourceName string, maxOpenConns int) (*sql.DB, error) {
	logger.Infof("connecting to [%s] database", driverName) // dataSource can contain a password

	id := driverName + dataSourceName
	var p *sql.DB
	d.mutex.RLock()
	p, ok := d.dbs[id]
	if ok {
		logger.Infof("reuse [%s] database (cached)", driverName)
		d.mutex.RUnlock()
		return p, nil
	}
	d.mutex.RUnlock()

	d.mutex.Lock()
	defer d.mutex.Unlock()

	// check again
	p, ok = d.dbs[id]
	if ok {
		return p, nil
	}
	p, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open db [%s]", driverName)
	}
	p.SetMaxOpenConns(maxOpenConns)
	d.dbs[id] = p
	return p, nil
}

type TtxDBDriver struct {
	*Driver
}

func (t *TtxDBDriver) Open(sp view.ServiceProvider, tmsID token.TMSID) (auditdbd.TokenTransactionDB, error) {
	return t.OpenTokenTransactionDB(sp, tmsID)
}

type TokenDBDriver struct {
	*Driver
}

func (t *TokenDBDriver) Open(sp view.ServiceProvider, tmsID token.TMSID) (auditdbd.TokenDB, error) {
	return t.OpenTokenDB(sp, tmsID)
}

type AuditDBDriver struct {
	*Driver
}

func (t *AuditDBDriver) Open(sp view.ServiceProvider, tmsID token.TMSID) (auditdbd.AuditTransactionDB, error) {
	return t.OpenAuditTransactionDB(sp, tmsID)
}

type IdentityDBDriver struct {
	*Driver
}

func (t *IdentityDBDriver) OpenWalletDB(sp view.ServiceProvider, tmsID token.TMSID) (auditdbd.WalletDB, error) {
	return t.Driver.OpenWalletDB(sp, tmsID)
}

func (t *IdentityDBDriver) OpenIdentityDB(sp view.ServiceProvider, tmsID token.TMSID) (auditdbd.IdentityDB, error) {
	return t.Driver.OpenIdentityDB(sp, tmsID)
}

func init() {
	root := NewDriver()
	ttxdb.Register("unity", &TtxDBDriver{Driver: root})
	tokendb.Register("unity", &TokenDBDriver{Driver: root})
	auditdb.Register("unity", &AuditDBDriver{Driver: root})
	identitydb.Register("unity", &IdentityDBDriver{Driver: root})
}
