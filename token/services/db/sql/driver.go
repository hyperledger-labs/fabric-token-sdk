/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
	"github.com/pkg/errors"
)

const sqlitePragmas = `
	PRAGMA journal_mode = WAL;
	PRAGMA busy_timeout = 5000;
	PRAGMA synchronous = NORMAL;
	PRAGMA cache_size = 1000000000;
	PRAGMA temp_store = memory;
	PRAGMA foreign_keys = ON;`

type Opts struct {
	Driver          string
	DataSource      string
	TablePrefix     string
	SkipCreateTable bool
	SkipPragmas     bool
	MaxOpenConns    int
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
			p, err := sql.Open(k.driverName, k.dataSourceName)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to open db [%s]", k.driverName)
			}
			if err := p.Ping(); err != nil {
				if strings.Contains(err.Error(), "out of memory (14)") {
					logger.Warnf("does the directory for the configured datasource exist?")
				}
				return nil, errors.Wrapf(err, "failed to open db [%s]", k.driverName)
			}

			// set some good defaults if the driver is sqlite
			if k.driverName == "sqlite" {
				if k.skipPragmas {
					if !strings.Contains(k.dataSourceName, "WAL") {
						logger.Warn("skipping default pragmas. Set at least ?_pragma=journal_mode(WAL) or similar in the dataSource to prevent SQLITE_BUSY errors")
					}
				} else {
					logger.Debug(sqlitePragmas)
					if _, err = p.Exec(sqlitePragmas); err != nil {
						return nil, fmt.Errorf("error setting pragmas: %w", err)
					}
				}
			}
			p.SetMaxOpenConns(k.maxOpenConns)

			return p, nil
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
	//opts.TablePrefix = db.EscapeForTableName(tmsID.Network, tmsID.Channel, tmsID.Namespace)
	return opts, nil
}

type dbKey struct {
	driverName, dataSourceName string
	maxOpenConns               int
	skipPragmas                bool
}

func (d *DBOpener) OpenSQLDB(driverName, dataSourceName string, maxOpenConns int, skipPragmas bool) (*sql.DB, error) {
	logger.Infof("connecting to [%s] database", driverName) // dataSource can contain a password

	return d.dbCache.Get(dbKey{driverName: driverName, dataSourceName: dataSourceName, maxOpenConns: maxOpenConns, skipPragmas: skipPragmas})
}

func key(k dbKey) string {
	return k.driverName + k.dataSourceName
}
