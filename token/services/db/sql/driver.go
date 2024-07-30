/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db"
	sql2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/postgres"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/sqlite"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pkg/errors"
	_ "modernc.org/sqlite"
)

type Opts struct {
	Driver          common.SQLDriverType
	DataSource      string
	TablePrefix     string
	SkipCreateTable bool
	SkipPragmas     bool
	MaxOpenConns    int
}

type NewDBOpts struct {
	DataSource   string
	TablePrefix  string
	CreateSchema bool
}

type DBOpener struct {
	dbCache   utils.LazyProvider[dbKey, *sql.DB]
	optsKey   string
	envVarKey string
}

func (d *DBOpener) Open(cp driver.ConfigProvider, tmsID token.TMSID) (*sql.DB, *Opts, error) {
	opts, err := d.compileOpts(cp, tmsID)
	if err != nil {
		return nil, nil, err
	}
	sqlDB, err := d.OpenSQLDB(opts.Driver, opts.DataSource, opts.MaxOpenConns, opts.SkipPragmas)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to open db at [%s:%s]", d.optsKey, d.envVarKey)
	}
	return sqlDB, opts, nil
}

func NewSQLDBOpener(optsKey, envVarKey string) *DBOpener {
	return &DBOpener{
		dbCache: utils.NewLazyProviderWithKeyMapper(key, func(k dbKey) (*sql.DB, error) {
			if k.driverName == sql2.SQLite {
				return sqlite.OpenDB(k.dataSourceName, k.maxOpenConns, k.skipPragmas)
			} else if k.driverName == sql2.Postgres {
				return postgres.OpenDB(k.dataSourceName, k.maxOpenConns)
			}

			return nil, errors.Errorf("unknown driver [%s]", k.driverName)
		}),
		optsKey:   optsKey,
		envVarKey: envVarKey,
	}
}

func (d *DBOpener) compileOpts(cp driver.ConfigProvider, tmsID token.TMSID) (*Opts, error) {
	opts := &Opts{}
	tmsConfig, err := config.NewService(cp).ConfigurationFor(tmsID.Network, tmsID.Channel, tmsID.Namespace)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to load configuration for tms [%s]", tmsID)
	}
	if err := tmsConfig.UnmarshalKey(d.optsKey, opts); err != nil {
		return nil, errors.Wrapf(err, "failed getting opts for vault")
	}
	if opts.Driver == "" {
		panic(fmt.Sprintf("%s.driver not set", d.optsKey))
	}

	dataSourceName := os.Getenv(d.envVarKey)
	if dataSourceName != "" {
		opts.DataSource = dataSourceName
	}
	if opts.DataSource == "" {
		return nil, errors.Errorf("either %s.dataSource in core.yaml or %s"+
			"environment variable must be set to a dataSourceName that can be used with the %s golang driver",
			d.optsKey, d.envVarKey, opts.Driver)
	}
	opts.TablePrefix = db.EscapeForTableName(tmsID.Network, tmsID.Channel, tmsID.Namespace)
	return opts, nil
}

type dbKey struct {
	driverName     common.SQLDriverType
	dataSourceName string
	maxOpenConns   int
	skipPragmas    bool
}

func (d *DBOpener) OpenSQLDB(driverName common.SQLDriverType, dataSourceName string, maxOpenConns int, skipPragmas bool) (*sql.DB, error) {
	logger.Infof("connecting to [%s] database", driverName) // dataSource can contain a password

	return d.dbCache.Get(dbKey{driverName: driverName, dataSourceName: dataSourceName, maxOpenConns: maxOpenConns, skipPragmas: skipPragmas})
}

func key(k dbKey) string {
	return string(k.driverName) + k.dataSourceName
}
