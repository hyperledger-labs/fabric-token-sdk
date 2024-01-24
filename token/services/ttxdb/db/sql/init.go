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
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb/driver"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("token-sdk.ttxdb.sql")

const (
	// OptsKey is the key for the opts in the config
	OptsKey   = "token.ttxdb.persistence.opts"
	EnvVarKey = "TTXDB_DATASOURCE"
)

type Opts struct {
	Driver       string
	DataSource   string
	TablePrefix  string
	CreateSchema bool
}

type Driver struct {
	mutex sync.RWMutex
	dbs   map[string]*sql.DB
}

func (d *Driver) Open(sp view.ServiceProvider, name string) (driver.TokenTransactionDB, error) {
	opts := &Opts{}
	if err := view.GetConfigService(sp).UnmarshalKey(OptsKey, opts); err != nil {
		return nil, errors.Wrapf(err, "failed getting opts for vault")
	}
	if opts.Driver == "" {
		panic(fmt.Sprintf("%s.driver not set. See https://github.com/golang/go/wiki/SQLDrivers", OptsKey))
	}

	dataSourceName := os.Getenv(EnvVarKey)
	if dataSourceName == "" {
		dataSourceName = opts.DataSource
	}
	if dataSourceName == "" {
		return nil, errors.Errorf("either %s.dataSource in core.yaml or %s"+
			"environment variable must be set to a dataSourceName that can be used with the %s golang driver",
			OptsKey, EnvVarKey, opts.Driver)
	}

	db, err := d.openDB(opts.Driver, dataSourceName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open db [%s]", opts.Driver)
	}
	if err = db.Ping(); err != nil {
		return nil, errors.Wrapf(err, "failed to ping db [%s]", opts.Driver)
	}

	tableNames, err := getTableNames(opts.TablePrefix, name)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get table names")
	}
	logger.Infof("connected to [%s:%s] database", opts.Driver, opts.TablePrefix)
	p := &Persistence{db: db, table: tableNames}
	if opts.CreateSchema {
		if err := p.CreateSchema(); err != nil {
			return nil, errors.Wrapf(err, "failed to create schema [%s:%s]", opts.Driver, tableNames)
		}
	}
	return p, nil
}

func (d *Driver) openDB(driverName, dataSourceName string) (*sql.DB, error) {
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

	d.mutex.Lock()
	defer d.mutex.Unlock()

	// check again
	p, ok = d.dbs[id]
	if ok {
		d.mutex.RUnlock()
		return p, nil
	}
	p, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open db [%s]", driverName)
	}
	d.dbs[id] = p
	return p, nil
}

func OpenDB(driverName, dataSourceName, tablePrefix, name string, createSchema bool) (driver.TokenTransactionDB, error) {
	logger.Infof("connecting to [%s:%s] database", driverName, tablePrefix) // dataSource can contain a password

	tableNames, err := getTableNames(tablePrefix, name)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get table names")
	}

	db, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open db [%s]", driverName)
	}
	if err = db.Ping(); err != nil {
		return nil, errors.Wrapf(err, "failed to ping db [%s]", driverName)
	}
	logger.Infof("connected to [%s:%s] database", driverName, tablePrefix)
	p := &Persistence{db: db, table: tableNames}
	if createSchema {
		if err := p.CreateSchema(); err != nil {
			return nil, errors.Wrapf(err, "failed to create schema [%s:%s]", driverName, tableNames)
		}
	}
	return p, nil
}

func init() {
	ttxdb.Register("sql", &Driver{dbs: make(map[string]*sql.DB)})
}
