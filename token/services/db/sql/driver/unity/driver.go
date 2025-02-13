/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package unity

import (
	sql2 "database/sql"

	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/lazy"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql"
	common2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/common"
	postgres2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/postgres"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/sqlite"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
	sql3 "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/driver/sql"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/postgres"
	sqlite2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/sqlite"
)

type rwDBs struct {
	readDB, writeDB *sql2.DB
}

const UnityPersistence driver2.PersistenceType = "unity"

type Driver struct {
	*sql3.Driver
}

func NewUnityDriver() driver.NamedDriver {
	var postgresDBCache lazy.Provider[common.Opts, *rwDBs] = lazy.NewProviderWithKeyMapper(key, func(opts common.Opts) (*rwDBs, error) {
		db, err := postgres2.OpenDB(opts.DataSource, opts.MaxOpenConns, opts.MaxIdleConns, opts.MaxIdleTime)
		return &rwDBs{readDB: db, writeDB: db}, err
	})
	var sqliteDBCache lazy.Provider[common.Opts, *rwDBs] = lazy.NewProviderWithKeyMapper(key, func(opts common.Opts) (*rwDBs, error) {
		readDB, writeDB, err := sqlite.OpenRWDBs(opts.DataSource, opts.MaxOpenConns, opts.MaxIdleConns, opts.MaxIdleTime, opts.SkipPragmas)
		return &rwDBs{readDB: readDB, writeDB: writeDB}, err
	})
	return driver.NamedDriver{
		Name: UnityPersistence,
		Driver: &Driver{
			Driver: &sql3.Driver{
				TokenLockCache: lazy.NewProviderWithKeyMapper(key, sql3.Combine(sql3.Openers[driver.TokenLockDB]{
					sql.SQLite:   newOpener(sqliteDBCache, sqlite2.NewTokenLockDB),
					sql.Postgres: newOpener(postgresDBCache, postgres.NewTokenLockDB),
				})),
				WalletCache: lazy.NewProviderWithKeyMapper(key, sql3.Combine(sql3.Openers[driver.WalletDB]{
					sql.SQLite:   newOpener(sqliteDBCache, sqlite2.NewWalletDB),
					sql.Postgres: newOpener(postgresDBCache, postgres.NewWalletDB),
				})),
				IdentityCache: lazy.NewProviderWithKeyMapper(key, sql3.Combine(sql3.Openers[driver.IdentityDB]{
					sql.SQLite:   newOpener(sqliteDBCache, sqlite2.NewIdentityDB),
					sql.Postgres: newOpener(postgresDBCache, postgres.NewIdentityDB),
				})),
				TokenCache: lazy.NewProviderWithKeyMapper(key, sql3.Combine(sql3.Openers[driver.TokenDB]{
					sql.SQLite:   newOpener(sqliteDBCache, sqlite2.NewTokenDB),
					sql.Postgres: newOpener(postgresDBCache, postgres.NewTokenDB),
				})),
				TokenNotifierCache: lazy.NewProviderWithKeyMapper(key, sql3.Combine(sql3.Openers[driver.TokenNotifier]{
					sql.SQLite:   newOpener(sqliteDBCache, sqlite2.NewTokenNotifier),
					sql.Postgres: newOpener(postgresDBCache, postgres.NewTokenNotifier),
				})),
				AuditTxCache: lazy.NewProviderWithKeyMapper(key, sql3.Combine(sql3.Openers[driver.AuditTransactionDB]{
					sql.SQLite:   newOpener(sqliteDBCache, sqlite2.NewAuditTransactionDB),
					sql.Postgres: newOpener(postgresDBCache, postgres.NewAuditTransactionDB),
				})),
				OwnerTxCache: lazy.NewProviderWithKeyMapper(key, sql3.Combine(sql3.Openers[driver.TokenTransactionDB]{
					sql.SQLite:   newOpener(sqliteDBCache, sqlite2.NewTransactionDB),
					sql.Postgres: newOpener(postgresDBCache, postgres.NewTransactionDB),
				})),
			},
		},
	}
}

func newOpener[T any](dbCache lazy.Provider[common.Opts, *rwDBs], newDB func(db *sql2.DB, opts common.NewDBOpts) (T, error)) sql3.Opener[T] {
	return func(opts common.Opts) (T, error) {
		dbs, err := dbCache.Get(opts)
		if err != nil {
			return utils.Zero[T](), err
		}
		return newDB(dbs.writeDB, common.NewDBOptsFromOpts(opts))
	}
}

func key(k common2.Opts) string {
	return string(k.Driver) + k.DataSource + k.TablePrefix
}
