/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"database/sql"
	"fmt"
	"os"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
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
	Parallelism  bool
}

type Driver struct {
}

func (d Driver) Open(sp view2.ServiceProvider, name string) (driver.TokenTransactionDB, error) {
	opts := &Opts{}
	if err := view2.GetConfigService(sp).UnmarshalKey(OptsKey, opts); err != nil {
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
	return OpenDB(opts.Driver, dataSourceName, opts.TablePrefix, name, opts.CreateSchema, opts.Parallelism)
}

func OpenDB(driverName, dataSourceName, tablePrefix, name string, createSchema, parallelism bool) (driver.TokenTransactionDB, error) {
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
	var ttxDB driver.TokenTransactionDB
	p := &Persistence{db: db, table: tableNames}
	ttxDB = p
	if !parallelism {
		ttxDB = &SerializedPersistence{Persistence: p}
	}
	if createSchema {
		if err := p.CreateSchema(); err != nil {
			return nil, errors.Wrapf(err, "failed to create schema [%s:%s]", driverName, tableNames)
		}
	}
	return ttxDB, nil
}

func init() {
	ttxdb.Register("sql", &Driver{})
}
