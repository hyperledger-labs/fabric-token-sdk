/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sqlite

import (
	"strings"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/cache/secondcache"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/lazy"
	fscSqlite "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/sqlite"
	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/common"
	
	driver3 "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/driver"
	common2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/common"
)

type configProvider interface {
	GetOpts(name driver2.PersistenceName, params ...string) (*fscSqlite.Config, error)
}

type Driver struct {
	cp configProvider

	TokenLock lazy.Provider[fscSqlite.Config, *TokenLockStore]
	Wallet    lazy.Provider[fscSqlite.Config, *WalletStore]
	Identity  lazy.Provider[fscSqlite.Config, *IdentityStore]
	Token     lazy.Provider[fscSqlite.Config, *TokenStore]
	AuditTx   lazy.Provider[fscSqlite.Config, *AuditTransactionStore]
	OwnerTx   lazy.Provider[fscSqlite.Config, *OwnerTransactionStore]
	KeyStore  lazy.Provider[fscSqlite.Config, *KeystoreStore]
}

func NewNamedDriver(config driver3.Config, dbProvider fscSqlite.DbProvider) driver3.NamedDriver {
	return driver3.NamedDriver{
		Name:   fscSqlite.Persistence,
		Driver: NewDriverWithDbProvider(config, dbProvider),
	}
}

func NewDriver(config driver3.Config) *Driver {
	return NewDriverWithDbProvider(config, fscSqlite.NewDbProvider())
}

func NewDriverWithDbProvider(config driver3.Config, dbProvider fscSqlite.DbProvider) *Driver {
	d := &Driver{
		cp: fscSqlite.NewConfigProvider(common.NewConfig(config)),
	}

	d.TokenLock = newProviderWithKeyMapper(dbProvider, NewTokenLockStore)
	d.Wallet = newProviderWithKeyMapper(dbProvider, NewWalletStore)
	d.Identity = newIdentityStoreProvider(dbProvider)
	d.Token = newProviderWithKeyMapper(dbProvider, NewTokenStore)
	d.AuditTx = newProviderWithKeyMapper(dbProvider, NewAuditTransactionStore)
	d.OwnerTx = newProviderWithKeyMapper(dbProvider, NewTransactionStore)
	d.KeyStore = newProviderWithKeyMapper(dbProvider, NewKeystoreStore)

	return d
}

func newIdentityStoreProvider(dbProvider fscSqlite.DbProvider) lazy.Provider[fscSqlite.Config, *IdentityStore] {
	return lazy.NewProviderWithKeyMapper(key, func(o fscSqlite.Config) (*IdentityStore, error) {
		opts := fscSqlite.Opts{
			DataSource:      o.DataSource,
			SkipPragmas:     o.SkipPragmas,
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
		tableNames, err := common2.GetTableNames(o.TablePrefix, o.TableNameParams...)
		if err != nil {
			return nil, err
		}

		p, err := common2.NewIdentityStoreWithNotifier(
			dbs.ReadDB,
			dbs.WriteDB,
			tableNames,
			secondcache.NewTyped[bool](5000),
			secondcache.NewTyped[[]byte](5000),
			NewConditionInterpreter(),
			&fscSqlite.ErrorMapper{},
			nil,
		)
		if err != nil {
			return nil, err
		}
		if !o.SkipCreateTable {
			if err := p.CreateSchema(); err != nil {
				return nil, err
			}
		}

		return p, nil
	})
}

func (d *Driver) NewTokenLock(name driver2.PersistenceName, params ...string) (driver3.TokenLockStore, error) {
	opts, err := d.cp.GetOpts(name, params...)
	if err != nil {
		return nil, err
	}

	return d.TokenLock.Get(*opts)
}

func (d *Driver) NewWallet(name driver2.PersistenceName, params ...string) (driver3.WalletStore, error) {
	opts, err := d.cp.GetOpts(name, params...)
	if err != nil {
		return nil, err
	}

	return d.Wallet.Get(*opts)
}

func (d *Driver) NewIdentity(name driver2.PersistenceName, params ...string) (driver3.IdentityStore, error) {
	opts, err := d.cp.GetOpts(name, params...)
	if err != nil {
		return nil, err
	}

	return d.Identity.Get(*opts)
}

func (d *Driver) NewKeyStore(name driver2.PersistenceName, params ...string) (driver3.KeyStore, error) {
	opts, err := d.cp.GetOpts(name, params...)
	if err != nil {
		return nil, err
	}

	return d.KeyStore.Get(*opts)
}

func (d *Driver) NewToken(name driver2.PersistenceName, params ...string) (driver3.TokenStore, error) {
	opts, err := d.cp.GetOpts(name, params...)
	if err != nil {
		return nil, err
	}

	return d.Token.Get(*opts)
}

func (d *Driver) NewAuditTransaction(name driver2.PersistenceName, params ...string) (driver3.AuditTransactionStore, error) {
	opts, err := d.cp.GetOpts(name, append(params, "aud")...)
	if err != nil {
		return nil, err
	}

	return d.AuditTx.Get(*opts)
}

func (d *Driver) NewOwnerTransaction(name driver2.PersistenceName, params ...string) (driver3.TokenTransactionStore, error) {
	opts, err := d.cp.GetOpts(name, params...)
	if err != nil {
		return nil, err
	}

	return d.OwnerTx.Get(*opts)
}

func newProviderWithKeyMapper[V common.DBObject](dbProvider fscSqlite.DbProvider, constructor common2.PersistenceConstructor[V]) lazy.Provider[fscSqlite.Config, V] {
	return lazy.NewProviderWithKeyMapper(key, func(o fscSqlite.Config) (V, error) {
		opts := fscSqlite.Opts{
			DataSource:      o.DataSource,
			SkipPragmas:     o.SkipPragmas,
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
		tableNames, err := common2.GetTableNames(o.TablePrefix, o.TableNameParams...)
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

func key(k fscSqlite.Config) string {
	return "sqlite" + k.DataSource + k.TablePrefix + strings.Join(k.TableNameParams, "_")
}
