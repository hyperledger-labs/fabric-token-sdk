/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"fmt"
	"os"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	sqldb "github.com/hyperledger-labs/fabric-token-sdk/token/services/sql"
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
	MaxOpenConns int
}

type Driver struct {
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

	return OpenDB(opts.Driver, opts.DataSource, opts.TablePrefix, name, opts.CreateSchema, opts.MaxOpenConns)
}

func OpenDB(driverName, dataSourceName, tablePrefix, name string, createSchema bool, maxOpenConns int) (driver.TokenTransactionDB, error) {
	// TODO: this only allows for one instance/'name'. Do we need more?
	if sqldb.Transactions == nil {
		logger.Infof("initializing database [%s, %s, %s, %v, %d]", driverName, tablePrefix, name, createSchema, maxOpenConns)
		if err := sqldb.Init(driverName, dataSourceName, tablePrefix, name, createSchema, maxOpenConns); err != nil {
			return nil, err
		}
	}
	return sqldb.Transactions, nil
}

func init() {
	ttxdb.Register("sql", &Driver{})
}
