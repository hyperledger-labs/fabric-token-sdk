/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package postgres

import (
	"strings"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/cache/secondcache"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/lazy"
	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/postgres"
	driver3 "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/driver"
	common3 "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/common"
)

// configProvider defines the interface for retrieving database configuration.
type configProvider interface {
	// GetOpts returns the Postgres configuration for the given persistence name and parameters.
	GetOpts(name driver2.PersistenceName, params ...string) (*postgres.Config, error)
}

// Driver implements the token storage driver for Postgres.
type Driver struct {
	cp configProvider

	// Lazy providers for various store types to ensure they are initialized only when needed.
	TokenLock lazy.Provider[postgres.Config, *TokenLockStore]
	Wallet    lazy.Provider[postgres.Config, *WalletStore]
	Identity  lazy.Provider[postgres.Config, *IdentityStore]
	Token     lazy.Provider[postgres.Config, *TokenStore]
	AuditTx   lazy.Provider[postgres.Config, *AuditTransactionStore]
	OwnerTx   lazy.Provider[postgres.Config, *TransactionStore]
	KeyStore  lazy.Provider[postgres.Config, *KeystoreStore]
}

// NewNamedDriver returns a NamedDriver for Postgres.
func NewNamedDriver(config driver3.Config, dbProvider postgres.DbProvider) driver3.NamedDriver {
	return driver3.NamedDriver{
		Name:   postgres.Persistence,
		Driver: NewDriverWithDbProvider(config, dbProvider),
	}
}

// NewDriver returns a new Driver for Postgres using the default database provider.
func NewDriver(config driver3.Config) *Driver {
	return NewDriverWithDbProvider(config, postgres.NewDbProvider())
}

// NewDriverWithDbProvider returns a new Driver for Postgres using the given database provider.
func NewDriverWithDbProvider(config driver3.Config, dbProvider postgres.DbProvider) *Driver {
	d := &Driver{
		cp: postgres.NewConfigProvider(common.NewConfig(config)),
	}

	d.TokenLock = newProviderWithKeyMapper(dbProvider, NewTokenLockStore)
	d.Wallet = newProviderWithKeyMapper(dbProvider, NewWalletStore)
	d.Identity = newIdentityStoreProvider(dbProvider)
	d.Token = newTokenStoreProvider(dbProvider)
	d.AuditTx = newProviderWithKeyMapper(dbProvider, NewAuditTransactionStore)
	d.OwnerTx = newProviderWithKeyMapper(dbProvider, NewTransactionStore)
	d.KeyStore = newProviderWithKeyMapper(dbProvider, NewKeystoreStore)

	return d
}

// newTokenStoreProvider returns a lazy provider for TokenStore.
func newTokenStoreProvider(dbProvider postgres.DbProvider) lazy.Provider[postgres.Config, *TokenStore] {
	return lazy.NewProviderWithKeyMapper(key, func(o postgres.Config) (*TokenStore, error) {
		opts := postgres.Opts{
			DataSource:      o.DataSource,
			MaxOpenConns:    o.MaxOpenConns,
			MaxIdleConns:    *o.MaxIdleConns,
			MaxIdleTime:     *o.MaxIdleTime,
			TablePrefix:     o.TablePrefix,
			TableNameParams: o.TableNameParams,
			Tracing:         o.Tracing,
		}
		dbs, err := dbProvider.Get(opts)
		if err != nil {
			return nil, err
		}
		tableNames, err := common3.GetTableNames(o.TablePrefix, o.TableNameParams...)
		if err != nil {
			return nil, err
		}

		// notifier
		notifier, err := NewTokenNotifier(dbs, tableNames, o.DataSource)
		if err != nil {
			return nil, err
		}

		// db
		p, err := NewTokenStoreWithNotifier(dbs, tableNames, notifier)
		if err != nil {
			return nil, err
		}
		if !o.SkipCreateTable {
			if err := p.CreateSchema(); err != nil {
				return nil, err
			}
			if err := notifier.CreateSchema(); err != nil {
				return nil, err
			}
		}

		return p, nil
	})
}

// newIdentityStoreProvider returns a lazy provider for IdentityStore.
func newIdentityStoreProvider(dbProvider postgres.DbProvider) lazy.Provider[postgres.Config, *IdentityStore] {
	return lazy.NewProviderWithKeyMapper(key, func(o postgres.Config) (*IdentityStore, error) {
		opts := postgres.Opts{
			DataSource:      o.DataSource,
			MaxOpenConns:    o.MaxOpenConns,
			MaxIdleConns:    *o.MaxIdleConns,
			MaxIdleTime:     *o.MaxIdleTime,
			TablePrefix:     o.TablePrefix,
			TableNameParams: o.TableNameParams,
			Tracing:         o.Tracing,
		}
		dbs, err := dbProvider.Get(opts)
		if err != nil {
			return nil, err
		}
		tableNames, err := common3.GetTableNames(o.TablePrefix, o.TableNameParams...)
		if err != nil {
			return nil, err
		}

		// notifier
		notifier, err := NewIdentityNotifier(dbs, tableNames, o.DataSource)
		if err != nil {
			return nil, err
		}

		// db
		p, err := common3.NewIdentityStoreWithNotifier(
			dbs.ReadDB,
			dbs.WriteDB,
			tableNames,
			secondcache.NewTyped[bool](5000),
			secondcache.NewTyped[[]byte](5000),
			postgres.NewConditionInterpreter(),
			&postgres.ErrorMapper{},
			notifier,
		)
		if err != nil {
			return nil, err
		}
		if !o.SkipCreateTable {
			if err := p.CreateSchema(); err != nil {
				return nil, err
			}
			if err := notifier.CreateSchema(); err != nil {
				return nil, err
			}
		}

		return p, nil
	})
}

// NewTokenLock returns a new TokenLockStore.
func (d *Driver) NewTokenLock(name driver2.PersistenceName, params ...string) (driver3.TokenLockStore, error) {
	opts, err := d.cp.GetOpts(name, params...)
	if err != nil {
		return nil, err
	}

	return d.TokenLock.Get(*opts)
}

// NewWallet returns a new WalletStore.
func (d *Driver) NewWallet(name driver2.PersistenceName, params ...string) (driver3.WalletStore, error) {
	opts, err := d.cp.GetOpts(name, params...)
	if err != nil {
		return nil, err
	}

	return d.Wallet.Get(*opts)
}

// NewIdentity returns a new IdentityStore.
func (d *Driver) NewIdentity(name driver2.PersistenceName, params ...string) (driver3.IdentityStore, error) {
	opts, err := d.cp.GetOpts(name, params...)
	if err != nil {
		return nil, err
	}

	return d.Identity.Get(*opts)
}

// NewKeyStore returns a new KeyStoreStore.
func (d *Driver) NewKeyStore(name driver2.PersistenceName, params ...string) (driver3.KeyStore, error) {
	opts, err := d.cp.GetOpts(name, params...)
	if err != nil {
		return nil, err
	}

	return d.KeyStore.Get(*opts)
}

// NewToken returns a new TokenStore.
func (d *Driver) NewToken(name driver2.PersistenceName, params ...string) (driver3.TokenStore, error) {
	opts, err := d.cp.GetOpts(name, params...)
	if err != nil {
		return nil, err
	}

	return d.Token.Get(*opts)
}

// NewAuditTransaction returns a new AuditTransactionStore.
func (d *Driver) NewAuditTransaction(name driver2.PersistenceName, params ...string) (driver3.AuditTransactionStore, error) {
	opts, err := d.cp.GetOpts(name, append(params, "aud")...)
	if err != nil {
		return nil, err
	}

	return d.AuditTx.Get(*opts)
}

// NewOwnerTransaction returns a new TokenTransactionStore.
func (d *Driver) NewOwnerTransaction(name driver2.PersistenceName, params ...string) (driver3.TokenTransactionStore, error) {
	opts, err := d.cp.GetOpts(name, params...)
	if err != nil {
		return nil, err
	}

	return d.OwnerTx.Get(*opts)
}

// newProviderWithKeyMapper returns a lazy provider for a DB object using a common constructor.
func newProviderWithKeyMapper[V common.DBObject](dbProvider postgres.DbProvider, constructor common3.PersistenceConstructor[V]) lazy.Provider[postgres.Config, V] {
	return lazy.NewProviderWithKeyMapper(key, func(o postgres.Config) (V, error) {
		opts := postgres.Opts{
			DataSource:      o.DataSource,
			MaxOpenConns:    o.MaxOpenConns,
			MaxIdleConns:    *o.MaxIdleConns,
			MaxIdleTime:     *o.MaxIdleTime,
			TablePrefix:     o.TablePrefix,
			TableNameParams: o.TableNameParams,
			Tracing:         o.Tracing,
		}
		dbs, err := dbProvider.Get(opts)
		if err != nil {
			return utils.Zero[V](), err
		}
		tableNames, err := common3.GetTableNames(o.TablePrefix, o.TableNameParams...)
		if err != nil {
			return utils.Zero[V](), err
		}
		p, err := constructor(dbs, tableNames)
		if err != nil {
			return utils.Zero[V](), err
		}
		if !o.SkipCreateTable {
			if err := p.CreateSchema(); err != nil {
				return utils.Zero[V](), err
			}
		}

		return p, nil
	})
}

// key returns a unique key for the given Postgres configuration.
func key(k postgres.Config) string {
	return "postgres" + k.DataSource + k.TablePrefix + strings.Join(k.TableNameParams, "_")
}
