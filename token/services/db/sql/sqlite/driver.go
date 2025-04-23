/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sqlite

import (
	"strings"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/lazy"
	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/sqlite"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
)

type configProvider interface {
	GetOpts(name driver2.PersistenceName, params ...string) (*sqlite.Config, error)
}

type Driver struct {
	cp configProvider

	TokenLockCache     lazy.Provider[sqlite.Config, *TokenLockDB]
	WalletCache        lazy.Provider[sqlite.Config, *WalletDB]
	IdentityCache      lazy.Provider[sqlite.Config, *IdentityDB]
	TokenCache         lazy.Provider[sqlite.Config, *TokenDB]
	TokenNotifierCache lazy.Provider[sqlite.Config, *TokenNotifier]
	AuditTxCache       lazy.Provider[sqlite.Config, *AuditTransactionDB]
	OwnerTxCache       lazy.Provider[sqlite.Config, *TransactionDB]
}

func NewNamedDriver(config driver.Config) driver.NamedDriver {
	return driver.NamedDriver{
		Name:   sqlite.Persistence,
		Driver: NewDriver(config),
	}
}

func NewDriver(config driver.Config) *Driver {
	return &Driver{
		cp: sqlite.NewConfigProvider(common.NewConfig(config)),

		TokenLockCache:     newProviderWithKeyMapper(NewTokenLockDB),
		WalletCache:        newProviderWithKeyMapper(NewWalletDB),
		IdentityCache:      newProviderWithKeyMapper(NewIdentityDB),
		TokenCache:         newProviderWithKeyMapper(NewTokenDB),
		TokenNotifierCache: newProviderWithKeyMapper(NewTokenNotifier),
		AuditTxCache:       newProviderWithKeyMapper(NewAuditTransactionDB),
		OwnerTxCache:       newProviderWithKeyMapper(NewTransactionDB),
	}
}

func (d *Driver) NewTokenLock(name driver2.PersistenceName, params ...string) (driver.TokenLockStore, error) {
	opts, err := d.cp.GetOpts(name, params...)
	if err != nil {
		return nil, err
	}
	return d.TokenLockCache.Get(*opts)
}

func (d *Driver) NewWallet(name driver2.PersistenceName, params ...string) (driver.WalletStore, error) {
	opts, err := d.cp.GetOpts(name, params...)
	if err != nil {
		return nil, err
	}
	return d.WalletCache.Get(*opts)
}

func (d *Driver) NewIdentity(name driver2.PersistenceName, params ...string) (driver.IdentityStore, error) {
	opts, err := d.cp.GetOpts(name, params...)
	if err != nil {
		return nil, err
	}
	return d.IdentityCache.Get(*opts)
}

func (d *Driver) NewToken(name driver2.PersistenceName, params ...string) (driver.TokenStore, error) {
	opts, err := d.cp.GetOpts(name, params...)
	if err != nil {
		return nil, err
	}

	return d.TokenCache.Get(*opts)
}

func (d *Driver) NewTokenNotifier(name driver2.PersistenceName, params ...string) (driver.TokenNotifier, error) {
	opts, err := d.cp.GetOpts(name, params...)
	if err != nil {
		return nil, err
	}
	return d.TokenNotifierCache.Get(*opts)
}

func (d *Driver) NewAuditTransaction(name driver2.PersistenceName, params ...string) (driver.AuditTransactionStore, error) {
	opts, err := d.cp.GetOpts(name, params...)
	if err != nil {
		return nil, err
	}
	return d.AuditTxCache.Get(*opts)
}

func (d *Driver) NewOwnerTransaction(name driver2.PersistenceName, params ...string) (driver.TokenTransactionStore, error) {
	opts, err := d.cp.GetOpts(name, params...)
	if err != nil {
		return nil, err
	}
	return d.OwnerTxCache.Get(*opts)
}

func newProviderWithKeyMapper[V common.DBObject](constructor common.PersistenceConstructor[sqlite.Opts, V]) lazy.Provider[sqlite.Config, V] {
	return lazy.NewProviderWithKeyMapper(key, func(o sqlite.Config) (V, error) {
		p, err := constructor(sqlite.Opts{
			DataSource:      o.DataSource,
			SkipPragmas:     o.SkipPragmas,
			MaxOpenConns:    o.MaxOpenConns,
			MaxIdleConns:    *o.MaxIdleConns,
			MaxIdleTime:     *o.MaxIdleTime,
			TablePrefix:     o.TablePrefix,
			TableNameParams: o.TableNameParams,
		})
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
