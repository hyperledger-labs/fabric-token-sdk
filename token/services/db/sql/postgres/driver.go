/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package postgres

import (
	"strings"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/lazy"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/postgres"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
)

type Driver struct {
	TokenLockCache     lazy.Provider[postgres.Config, *TokenLockDB]
	WalletCache        lazy.Provider[postgres.Config, *WalletDB]
	IdentityCache      lazy.Provider[postgres.Config, *IdentityDB]
	TokenCache         lazy.Provider[postgres.Config, *TokenDB]
	TokenNotifierCache lazy.Provider[postgres.Config, *TokenNotifier]
	AuditTxCache       lazy.Provider[postgres.Config, *AuditTransactionDB]
	OwnerTxCache       lazy.Provider[postgres.Config, *TransactionDB]
}

func NewNamedDriver() driver.NamedDriver {
	return driver.NamedDriver{
		Name:   postgres.Persistence,
		Driver: NewDriver(),
	}
}

func NewDriver() *Driver {
	return &Driver{
		TokenLockCache:     newProviderWithKeyMapper(NewTokenLockDB),
		WalletCache:        newProviderWithKeyMapper(NewWalletDB),
		IdentityCache:      newProviderWithKeyMapper(NewIdentityDB),
		TokenCache:         newProviderWithKeyMapper(NewTokenDB),
		TokenNotifierCache: newProviderWithKeyMapper(NewTokenNotifier),
		AuditTxCache:       newProviderWithKeyMapper(NewAuditTransactionDB),
		OwnerTxCache:       newProviderWithKeyMapper(NewTransactionDB),
	}
}

func (d *Driver) NewTokenLock(cfg driver.Config, params ...string) (driver.TokenLockDB, error) {
	opts, err := postgres.NewConfigProvider(cfg).GetOpts(params...)
	if err != nil {
		return nil, err
	}
	return d.TokenLockCache.Get(*opts)
}

func (d *Driver) NewWallet(cfg driver.Config, params ...string) (driver.WalletDB, error) {
	opts, err := postgres.NewConfigProvider(cfg).GetOpts(params...)
	if err != nil {
		return nil, err
	}
	return d.WalletCache.Get(*opts)
}

func (d *Driver) NewIdentity(cfg driver.Config, params ...string) (driver.IdentityDB, error) {
	opts, err := postgres.NewConfigProvider(cfg).GetOpts(params...)
	if err != nil {
		return nil, err
	}
	return d.IdentityCache.Get(*opts)
}

func (d *Driver) NewToken(cfg driver.Config, params ...string) (driver.TokenDB, error) {
	opts, err := postgres.NewConfigProvider(cfg).GetOpts(params...)
	if err != nil {
		return nil, err
	}
	return d.TokenCache.Get(*opts)
}

func (d *Driver) NewTokenNotifier(cfg driver.Config, params ...string) (driver.TokenNotifier, error) {
	opts, err := postgres.NewConfigProvider(cfg).GetOpts(params...)
	if err != nil {
		return nil, err
	}
	return d.TokenNotifierCache.Get(*opts)
}

func (d *Driver) NewAuditTransaction(cfg driver.Config, params ...string) (driver.AuditTransactionDB, error) {
	opts, err := postgres.NewConfigProvider(cfg).GetOpts(params...)
	if err != nil {
		return nil, err
	}
	return d.AuditTxCache.Get(*opts)
}

func (d *Driver) NewOwnerTransaction(cfg driver.Config, params ...string) (driver.TokenTransactionDB, error) {
	opts, err := postgres.NewConfigProvider(cfg).GetOpts(params...)
	if err != nil {
		return nil, err
	}
	return d.OwnerTxCache.Get(*opts)
}

func newProviderWithKeyMapper[V common.DBObject](constructor common.PersistenceConstructor[postgres.Opts, V]) lazy.Provider[postgres.Config, V] {
	return lazy.NewProviderWithKeyMapper(key, func(o postgres.Config) (V, error) {
		p, err := constructor(postgres.Opts{
			DataSource:      o.DataSource,
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

func key(k postgres.Config) string {
	return "postgres" + k.DataSource + k.TablePrefix + strings.Join(k.TableNameParams, "_")
}
