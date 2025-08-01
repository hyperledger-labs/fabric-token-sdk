/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package postgres

import (
	"context"
	"strings"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/lazy"
	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/postgres"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	common3 "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
)

type configProvider interface {
	GetOpts(name driver2.PersistenceName, params ...string) (*postgres.Config, error)
}

type Driver struct {
	cp configProvider

	TokenLock     lazy.Provider[postgres.Config, *TokenLockStore]
	Wallet        lazy.Provider[postgres.Config, *WalletStore]
	Identity      lazy.Provider[postgres.Config, *IdentityStore]
	Token         lazy.Provider[postgres.Config, *TokenStore]
	TokenNotifier lazy.Provider[postgres.Config, *TokenNotifier]
	AuditTx       lazy.Provider[postgres.Config, *AuditTransactionStore]
	OwnerTx       lazy.Provider[postgres.Config, *TransactionStore]
}

func NewNamedDriver(config driver.Config, dbProvider postgres.DbProvider) driver.NamedDriver {
	return driver.NamedDriver{
		Name:   postgres.Persistence,
		Driver: NewDriverWithDbProvider(config, dbProvider),
	}
}

func NewDriver(config driver.Config) *Driver {
	return NewDriverWithDbProvider(config, postgres.NewDbProvider())
}

func NewDriverWithDbProvider(config driver.Config, dbProvider postgres.DbProvider) *Driver {
	return &Driver{
		cp: postgres.NewConfigProvider(common.NewConfig(config)),

		TokenLock:     newProviderWithKeyMapper(dbProvider, NewTokenLockStore),
		Wallet:        newProviderWithKeyMapper(dbProvider, NewWalletStore),
		Identity:      newProviderWithKeyMapper(dbProvider, NewIdentityStore),
		Token:         newProviderWithKeyMapper(dbProvider, NewTokenStore),
		TokenNotifier: newTokenNotifierProvider(dbProvider),
		AuditTx:       newProviderWithKeyMapper(dbProvider, NewAuditTransactionStore),
		OwnerTx:       newProviderWithKeyMapper(dbProvider, NewTransactionStore),
	}
}

func (d *Driver) NewTokenLock(ctx context.Context, name driver2.PersistenceName, params ...string) (driver.TokenLockStore, error) {
	opts, err := d.cp.GetOpts(name, params...)
	if err != nil {
		return nil, err
	}
	return d.TokenLock.Get(*opts)
}

func (d *Driver) NewWallet(ctx context.Context, name driver2.PersistenceName, params ...string) (driver.WalletStore, error) {
	opts, err := d.cp.GetOpts(name, params...)
	if err != nil {
		return nil, err
	}
	return d.Wallet.Get(*opts)
}

func (d *Driver) NewIdentity(ctx context.Context, name driver2.PersistenceName, params ...string) (driver.IdentityStore, error) {
	opts, err := d.cp.GetOpts(name, params...)
	if err != nil {
		return nil, err
	}
	return d.Identity.Get(*opts)
}

func (d *Driver) NewToken(ctx context.Context, name driver2.PersistenceName, params ...string) (driver.TokenStore, error) {
	opts, err := d.cp.GetOpts(name, params...)
	if err != nil {
		return nil, err
	}
	return d.Token.Get(*opts)
}

func (d *Driver) NewTokenNotifier(ctx context.Context, name driver2.PersistenceName, params ...string) (driver.TokenNotifier, error) {
	opts, err := d.cp.GetOpts(name, params...)
	if err != nil {
		return nil, err
	}
	return d.TokenNotifier.Get(*opts)
}

func (d *Driver) NewAuditTransaction(ctx context.Context, name driver2.PersistenceName, params ...string) (driver.AuditTransactionStore, error) {
	opts, err := d.cp.GetOpts(name, append(params, "aud")...)
	if err != nil {
		return nil, err
	}
	return d.AuditTx.Get(*opts)
}

func (d *Driver) NewOwnerTransaction(ctx context.Context, name driver2.PersistenceName, params ...string) (driver.TokenTransactionStore, error) {
	opts, err := d.cp.GetOpts(name, params...)
	if err != nil {
		return nil, err
	}
	return d.OwnerTx.Get(*opts)
}

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

func newTokenNotifierProvider(dbProvider postgres.DbProvider) lazy.Provider[postgres.Config, *TokenNotifier] {
	return lazy.NewProviderWithKeyMapper(key, func(o postgres.Config) (*TokenNotifier, error) {
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
		p, err := NewTokenNotifier(dbs, tableNames, o.DataSource)
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

func key(k postgres.Config) string {
	return "postgres" + k.DataSource + k.TablePrefix + strings.Join(k.TableNameParams, "_")
}
