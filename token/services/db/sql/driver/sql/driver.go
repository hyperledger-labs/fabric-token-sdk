/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	sql2 "database/sql"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/lazy"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/common"
	postgres2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/postgres"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/sqlite"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	common2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/postgres"
	sqlite2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/sqlite"
	"github.com/pkg/errors"
)

type Driver struct {
	TokenLockCache     lazy.Provider[common2.Opts, driver.TokenLockDB]
	WalletCache        lazy.Provider[common2.Opts, driver.WalletDB]
	IdentityCache      lazy.Provider[common2.Opts, driver.IdentityDB]
	TokenCache         lazy.Provider[common2.Opts, driver.TokenDB]
	TokenNotifierCache lazy.Provider[common2.Opts, driver.TokenNotifier]
	AuditTxCache       lazy.Provider[common2.Opts, driver.AuditTransactionDB]
	OwnerTxCache       lazy.Provider[common2.Opts, driver.TokenTransactionDB]
}

func NewDriver() driver.NamedDriver {
	return driver.NamedDriver{
		Name: sql.SQLPersistence,
		Driver: &Driver{
			TokenLockCache: lazy.NewProviderWithKeyMapper(key, Combine(Openers[driver.TokenLockDB]{
				sql.SQLite:   newSqliteOpener(sqlite2.NewTokenLockDB),
				sql.Postgres: newPostgresOpener(postgres.NewTokenLockDB),
			})),
			WalletCache: lazy.NewProviderWithKeyMapper(key, Combine(Openers[driver.WalletDB]{
				sql.SQLite:   newSqliteOpener(sqlite2.NewWalletDB),
				sql.Postgres: newPostgresOpener(postgres.NewWalletDB),
			})),
			IdentityCache: lazy.NewProviderWithKeyMapper(key, Combine(Openers[driver.IdentityDB]{
				sql.SQLite:   newSqliteOpener(sqlite2.NewIdentityDB),
				sql.Postgres: newPostgresOpener(postgres.NewIdentityDB),
			})),
			TokenCache: lazy.NewProviderWithKeyMapper(key, Combine(Openers[driver.TokenDB]{
				sql.SQLite:   newSqliteOpener(sqlite2.NewTokenDB),
				sql.Postgres: newPostgresOpener(postgres.NewTokenDB),
			})),
			TokenNotifierCache: lazy.NewProviderWithKeyMapper(key, Combine(Openers[driver.TokenNotifier]{
				sql.SQLite:   newSqliteOpener(sqlite2.NewTokenNotifier),
				sql.Postgres: newPostgresOpener(postgres.NewTokenNotifier),
			})),
			AuditTxCache: lazy.NewProviderWithKeyMapper(key, Combine(Openers[driver.AuditTransactionDB]{
				sql.SQLite:   newSqliteOpener(sqlite2.NewAuditTransactionDB),
				sql.Postgres: newPostgresOpener(postgres.NewAuditTransactionDB),
			})),
			OwnerTxCache: lazy.NewProviderWithKeyMapper(key, Combine(Openers[driver.TokenTransactionDB]{
				sql.SQLite:   newSqliteOpener(sqlite2.NewTransactionDB),
				sql.Postgres: newPostgresOpener(postgres.NewTransactionDB),
			})),
		},
	}
}

type Opener[T any] func(opts common2.Opts) (T, error)
type Openers[T any] map[common.SQLDriverType]Opener[T]

func Combine[T any](openers map[common.SQLDriverType]Opener[T]) Opener[T] {
	return func(opts common2.Opts) (T, error) {
		if constructor, ok := openers[opts.Driver]; !ok {
			return utils.Zero[T](), errors.New("driver not found")
		} else {
			return constructor(opts)
		}
	}
}

func newPostgresOpener[T any](newDB func(db *sql2.DB, opts common2.NewDBOpts) (T, error)) Opener[T] {
	return func(opts common2.Opts) (T, error) {
		return OpenPostgres[T](opts, newDB)
	}
}

func OpenPostgres[T any](opts common2.Opts, newDB func(db *sql2.DB, opts common2.NewDBOpts) (T, error)) (T, error) {
	readWriteDB, err := postgres2.OpenDB(opts.DataSource, opts.MaxOpenConns, opts.MaxIdleConns, opts.MaxIdleTime)
	if err != nil {
		return utils.Zero[T](), err
	}
	return newDB(readWriteDB, common2.NewDBOptsFromOpts(opts))
}

func newSqliteOpener[T any](newDB func(db *sql2.DB, opts common2.NewDBOpts) (T, error)) Opener[T] {
	return func(opts common2.Opts) (T, error) {
		return OpenSqlite(opts, newDB)
	}
}

func OpenSqlite[T any](opts common2.Opts, newDB func(db *sql2.DB, dbOpts common2.NewDBOpts) (T, error)) (T, error) {
	_, writeDB, err := sqlite.OpenRWDBs(opts.DataSource, opts.MaxOpenConns, opts.MaxIdleConns, opts.MaxIdleTime, opts.SkipPragmas)
	if err != nil {
		return utils.Zero[T](), err
	}
	return newDB(writeDB, common2.NewDBOptsFromOpts(opts))
}

func (d *Driver) NewTokenLock(opts common.Opts) (driver.TokenLockDB, error) {
	return d.TokenLockCache.Get(opts)
}

func (d *Driver) NewWallet(opts common.Opts) (driver.WalletDB, error) {
	return d.WalletCache.Get(opts)
}

func (d *Driver) NewIdentity(opts common.Opts) (driver.IdentityDB, error) {
	return d.IdentityCache.Get(opts)
}

func (d *Driver) NewToken(opts common.Opts) (driver.TokenDB, error) {
	return d.TokenCache.Get(opts)
}

func (d *Driver) NewTokenNotifier(opts common.Opts) (driver.TokenNotifier, error) {
	return d.TokenNotifierCache.Get(opts)
}

func (d *Driver) NewAuditTransaction(opts common.Opts) (driver.AuditTransactionDB, error) {
	return d.AuditTxCache.Get(opts)
}

func (d *Driver) NewOwnerTransaction(opts common.Opts) (driver.TokenTransactionDB, error) {
	return d.OwnerTxCache.Get(opts)
}

func key(k common2.Opts) string {
	return string(k.Driver) + k.DataSource + k.TablePrefix
}
