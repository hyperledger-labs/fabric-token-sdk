/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sqlite

import (
	"strings"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/lazy"
	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/sqlite"
	driver3 "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/driver"
	common2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/common"
)

type configProvider interface {
	GetOpts(name driver2.PersistenceName, params ...string) (*sqlite.Config, error)
}

type Driver struct {
	cp configProvider

	TokenLock     lazy.Provider[sqlite.Config, *TokenLockStore]
	Wallet        lazy.Provider[sqlite.Config, *WalletStore]
	Identity      lazy.Provider[sqlite.Config, *IdentityStore]
	Token         lazy.Provider[sqlite.Config, *TokenStore]
	TokenNotifier lazy.Provider[sqlite.Config, *TokenNotifier]
	AuditTx       lazy.Provider[sqlite.Config, *AuditTransactionStore]
	OwnerTx       lazy.Provider[sqlite.Config, *OwnerTransactionStore]
	KeyStore      lazy.Provider[sqlite.Config, *KeystoreStore]
}

func NewNamedDriver(config driver3.Config, dbProvider sqlite.DbProvider) driver3.NamedDriver {
	return driver3.NamedDriver{
		Name:   sqlite.Persistence,
		Driver: NewDriverWithDbProvider(config, dbProvider),
	}
}

func NewDriver(config driver3.Config) *Driver {
	return NewDriverWithDbProvider(config, sqlite.NewDbProvider())
}

func NewDriverWithDbProvider(config driver3.Config, dbProvider sqlite.DbProvider) *Driver {
	return &Driver{
		cp: sqlite.NewConfigProvider(common.NewConfig(config)),

		TokenLock:     newProviderWithKeyMapper(dbProvider, NewTokenLockStore),
		Wallet:        newProviderWithKeyMapper(dbProvider, NewWalletStore),
		Identity:      newProviderWithKeyMapper(dbProvider, NewIdentityStore),
		Token:         newProviderWithKeyMapper(dbProvider, NewTokenStore),
		TokenNotifier: newProviderWithKeyMapper(dbProvider, NewTokenNotifier),
		AuditTx:       newProviderWithKeyMapper(dbProvider, NewAuditTransactionStore),
		OwnerTx:       newProviderWithKeyMapper(dbProvider, NewTransactionStore),
		KeyStore:      newProviderWithKeyMapper(dbProvider, NewKeystoreStore),
	}
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

func (d *Driver) NewTokenNotifier(name driver2.PersistenceName, params ...string) (driver3.TokenNotifier, error) {
	opts, err := d.cp.GetOpts(name, params...)
	if err != nil {
		return nil, err
	}
	return d.TokenNotifier.Get(*opts)
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

func newProviderWithKeyMapper[V common.DBObject](dbProvider sqlite.DbProvider, constructor common2.PersistenceConstructor[V]) lazy.Provider[sqlite.Config, V] {
	return lazy.NewProviderWithKeyMapper(key, func(o sqlite.Config) (V, error) {
		opts := sqlite.Opts{
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

func key(k sqlite.Config) string {
	return "sqlite" + k.DataSource + k.TablePrefix + strings.Join(k.TableNameParams, "_")
}
