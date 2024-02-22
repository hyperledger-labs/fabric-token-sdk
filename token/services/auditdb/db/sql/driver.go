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

	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	sqldb "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("token-sdk.auditdb.sql")

const (
	// OptsKey is the key for the opts in the config
	OptsKey   = "auditdb.persistence.opts"
	EnvVarKey = "AUDITDB_DATASOURCE"
)

type Opts struct {
	Driver       string
	DataSource   string
	TablePrefix  string
	CreateSchema bool
	MaxOpenConns int
}

type Driver struct {
	mutex sync.RWMutex
	dbs   map[string]*sql.DB
}

func NewDriver() *Driver {
	return &Driver{dbs: make(map[string]*sql.DB)}
}

func (d *Driver) Open(sp view.ServiceProvider, tmsID token.TMSID) (driver.AuditTransactionDB, error) {
	opts := &Opts{}
	tmsConfig, err := config.NewTokenSDK(view.GetConfigService(sp)).GetTMS(tmsID.Network, tmsID.Channel, tmsID.Namespace)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to load configuration for tms [%s]", tmsID)
	}
	if err := tmsConfig.UnmarshalKey(OptsKey, opts); err != nil {
		return nil, errors.Wrapf(err, "failed getting opts for vault")
	}
	if opts.Driver == "" {
		panic(fmt.Sprintf("%s.driver not set. See https://github.com/golang/go/wiki/SQLDrivers", OptsKey))
	}

	name := sqldb.DatasourceName(tmsID)
	dataSourceName := os.Getenv(EnvVarKey)
	if dataSourceName == "" {
		dataSourceName = opts.DataSource
	}
	if dataSourceName == "" {
		return nil, errors.Errorf("either %s.dataSource in core.yaml or %s"+
			"environment variable must be set to a dataSourceName that can be used with the %s golang driver",
			OptsKey, EnvVarKey, opts.Driver)
	}

	sqlDB, err := d.OpenSQLDB(opts.Driver, opts.DataSource, opts.MaxOpenConns)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open db at [%s:%s:%s]", OptsKey, EnvVarKey, opts.Driver)
	}

	return sqldb.NewTransactionDB(sqlDB, opts.TablePrefix, name, opts.CreateSchema)
}

func (d *Driver) OpenSQLDB(driverName, dataSourceName string, maxOpenConns int) (*sql.DB, error) {
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

func init() {
	auditdb.Register("sql", NewDriver())
}
