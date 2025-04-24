/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package postgres

import (
	"strings"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/lazy"
	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/postgres"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
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

func NewNamedDriver(config driver.Config) driver.NamedDriver {
	return driver.NamedDriver{
		Name:   postgres.Persistence,
		Driver: NewDriver(config),
	}
}

func NewDriver(config driver.Config) *Driver {
	return &Driver{
		cp: postgres.NewConfigProvider(common.NewConfig(config)),

		TokenLock:     newProviderWithKeyMapper(NewTokenLockStore),
		Wallet:        newProviderWithKeyMapper(NewWalletStore),
		Identity:      newProviderWithKeyMapper(NewIdentityStore),
		Token:         newProviderWithKeyMapper(NewTokenStore),
		TokenNotifier: newProviderWithKeyMapper(NewTokenNotifier),
		AuditTx:       newProviderWithKeyMapper(NewAuditTransactionStore),
		OwnerTx:       newProviderWithKeyMapper(NewTransactionStore),
	}
}

func (d *Driver) NewTokenLock(name driver2.PersistenceName, params ...string) (driver.TokenLockStore, error) {
	opts, err := d.cp.GetOpts(name, params...)
	if err != nil {
		return nil, err
	}
	return d.TokenLock.Get(*opts)
}

func (d *Driver) NewWallet(name driver2.PersistenceName, params ...string) (driver.WalletStore, error) {
	opts, err := d.cp.GetOpts(name, params...)
	if err != nil {
		return nil, err
	}
	return d.Wallet.Get(*opts)
}

func (d *Driver) NewIdentity(name driver2.PersistenceName, params ...string) (driver.IdentityStore, error) {
	opts, err := d.cp.GetOpts(name, params...)
	if err != nil {
		return nil, err
	}
	return d.Identity.Get(*opts)
}

func (d *Driver) NewToken(name driver2.PersistenceName, params ...string) (driver.TokenStore, error) {
	opts, err := d.cp.GetOpts(name, params...)
	if err != nil {
		return nil, err
	}
	return d.Token.Get(*opts)
}

func (d *Driver) NewTokenNotifier(name driver2.PersistenceName, params ...string) (driver.TokenNotifier, error) {
	opts, err := d.cp.GetOpts(name, params...)
	if err != nil {
		return nil, err
	}
	return d.TokenNotifier.Get(*opts)
}

func (d *Driver) NewAuditTransaction(name driver2.PersistenceName, params ...string) (driver.AuditTransactionStore, error) {
	opts, err := d.cp.GetOpts(name, params...)
	if err != nil {
		return nil, err
	}
	return d.AuditTx.Get(*opts)
}

func (d *Driver) NewOwnerTransaction(name driver2.PersistenceName, params ...string) (driver.TokenTransactionStore, error) {
	opts, err := d.cp.GetOpts(name, params...)
	if err != nil {
		return nil, err
	}
	return d.OwnerTx.Get(*opts)
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
