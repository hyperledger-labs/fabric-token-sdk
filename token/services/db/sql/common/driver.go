/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"database/sql"

	utils2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/utils"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/lazy"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db"
	sql2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/postgres"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/sqlite"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pkg/errors"
	_ "modernc.org/sqlite"
)

type Opts = common.Opts

type OpenDBFunc[V any] func(k Opts) (V, error)

type NewDBFunc[V any] func(db *sql.DB, opts NewDBOpts) (V, error)

type NewDBOpts struct {
	DataSource   string
	TablePrefix  string
	CreateSchema bool
}

type Opener[V any] struct {
	dbCache   lazy.Provider[Opts, V]
	optsKey   string
	envVarKey string
}

type DBOpener = Opener[*sql.DB]

func NewDBOptsFromOpts(o Opts) NewDBOpts {
	return NewDBOpts{
		DataSource:   o.DataSource,
		TablePrefix:  o.TablePrefix,
		CreateSchema: !o.SkipCreateTable,
	}
}

func NewOpenerFromMap[V any](optsKey, envVarKey string, constructors map[common.SQLDriverType]OpenDBFunc[V]) *Opener[V] {
	return NewOpener[V](optsKey, envVarKey, func(opts Opts) (V, error) {
		if c, ok := constructors[opts.Driver]; ok {
			return c(opts)
		}
		panic("driver not supported")
	})
}

func NewOpener[V any](optsKey, envVarKey string, constructors OpenDBFunc[V]) *Opener[V] {
	return &Opener[V]{
		dbCache:   lazy.NewProviderWithKeyMapper(key, constructors),
		optsKey:   optsKey,
		envVarKey: envVarKey,
	}
}

func NewSQLDBOpener(optsKey, envVarKey string) *DBOpener {
	return NewOpenerFromMap[*sql.DB](optsKey, envVarKey, map[common.SQLDriverType]OpenDBFunc[*sql.DB]{
		sql2.SQLite: func(k Opts) (*sql.DB, error) {
			return sqlite.OpenDB(k.DataSource, k.MaxOpenConns, k.SkipPragmas)
		},
		sql2.Postgres: func(k Opts) (*sql.DB, error) { return postgres.OpenDB(k.DataSource, k.MaxOpenConns) },
	})
}

func (d *Opener[V]) Open(cp driver.ConfigProvider, tmsID token.TMSID) (V, error) {
	v, _, err := d.OpenWithOpts(cp, tmsID)
	return v, err
}

func (d *Opener[V]) OpenWithOpts(cp driver.ConfigProvider, tmsID token.TMSID) (V, *Opts, error) {
	opts, err := d.compileOpts(cp, tmsID)
	if err != nil {
		return utils2.Zero[V](), nil, err
	}
	sqlDB, err := d.dbCache.Get(*opts)
	if err != nil {
		return utils2.Zero[V](), nil, errors.Wrapf(err, "failed to open db at [%s:%s]", d.optsKey, d.envVarKey)
	}
	return sqlDB, opts, nil
}

func (d *Opener[V]) compileOpts(cp driver.ConfigProvider, tmsID token.TMSID) (*Opts, error) {
	tmsConfig, err := config.NewService(cp).ConfigurationFor(tmsID.Network, tmsID.Channel, tmsID.Namespace)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to load configuration for tms [%s]", tmsID)
	}
	opts, err := common.GetOpts(tmsConfig, d.optsKey, d.envVarKey)
	if err != nil {
		return nil, err
	}
	opts.TablePrefix = db.EscapeForTableName(tmsID.Network, tmsID.Channel, tmsID.Namespace)
	return opts, nil
}

func (d *Opener[V]) OpenSQLDB(driverName common.SQLDriverType, dataSourceName string, maxOpenConns int, skipPragmas bool) (V, error) {
	logger.Infof("connecting to [%s] database", driverName) // dataSource can contain a password

	return d.dbCache.Get(Opts{Driver: driverName, DataSource: dataSourceName, MaxOpenConns: maxOpenConns, SkipPragmas: skipPragmas})
}

func key(k Opts) string {
	return string(k.Driver) + k.DataSource
}
