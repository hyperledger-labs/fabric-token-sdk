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
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/config"
	"github.com/pkg/errors"
)

type Opts struct {
	Driver       string
	DataSource   string
	TablePrefix  string
	CreateSchema bool
	MaxOpenConns int
}

type DBOpener struct {
	mutex     sync.RWMutex
	dbs       map[string]*sql.DB
	optsKey   string
	envVarKey string
}

func (d *DBOpener) Open(sp view.ServiceProvider, tmsID token.TMSID) (*sql.DB, *Opts, error) {
	opts, err := d.compileOpts(sp, tmsID)
	if err != nil {
		return nil, nil, err
	}
	sqlDB, err := d.OpenSQLDB(opts.Driver, opts.DataSource, opts.MaxOpenConns)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to open db at [%s:%s:%s]", d.optsKey, d.envVarKey, opts.Driver)
	}
	return sqlDB, opts, nil
}

func NewSQLDBOpener(optsKey, envVarKey string) *DBOpener {
	return &DBOpener{
		dbs:       map[string]*sql.DB{},
		optsKey:   optsKey,
		envVarKey: envVarKey,
	}
}

func (d *DBOpener) compileOpts(sp view.ServiceProvider, tmsID token.TMSID) (*Opts, error) {
	opts := &Opts{}
	tmsConfig, err := config.NewTokenSDK(view.GetConfigService(sp)).GetTMS(tmsID.Network, tmsID.Channel, tmsID.Namespace)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to load configuration for tms [%s]", tmsID)
	}
	if err := tmsConfig.UnmarshalKey(d.optsKey, opts); err != nil {
		return nil, errors.Wrapf(err, "failed getting opts for vault")
	}
	if opts.Driver == "" {
		panic(fmt.Sprintf("%s.driver not set. See https://github.com/golang/go/wiki/SQLDrivers", d.optsKey))
	}

	dataSourceName := os.Getenv(d.envVarKey)
	if dataSourceName == "" {
		dataSourceName = opts.DataSource
	}
	if dataSourceName == "" {
		return nil, errors.Errorf("either %s.dataSource in core.yaml or %s"+
			"environment variable must be set to a dataSourceName that can be used with the %s golang driver",
			d.optsKey, d.envVarKey, opts.Driver)
	}
	return opts, nil
}

func (d *DBOpener) OpenSQLDB(driverName, dataSourceName string, maxOpenConns int) (*sql.DB, error) {
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
