/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package memory

import (
	mem "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/memory"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	sqlite2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/sqlite"
)

type Driver sqlite2.Driver

func NewNamedDriver() driver.NamedDriver {
	return driver.NamedDriver{
		Name:   mem.Persistence,
		Driver: NewDriver(),
	}
}

func NewDriver() *Driver {
	return (*Driver)(sqlite2.NewDriver())
}

func (d *Driver) NewTokenLock(_ driver.Config, params ...string) (driver.TokenLockDB, error) {
	return d.TokenLockCache.Get(mem.Op.GetConfig(params...))
}

func (d *Driver) NewWallet(_ driver.Config, params ...string) (driver.WalletDB, error) {
	return d.WalletCache.Get(mem.Op.GetConfig(params...))
}

func (d *Driver) NewIdentity(_ driver.Config, params ...string) (driver.IdentityDB, error) {
	return d.IdentityCache.Get(mem.Op.GetConfig(params...))
}

func (d *Driver) NewToken(_ driver.Config, params ...string) (driver.TokenDB, error) {
	return d.TokenCache.Get(mem.Op.GetConfig(params...))
}

func (d *Driver) NewTokenNotifier(_ driver.Config, params ...string) (driver.TokenNotifier, error) {
	return d.TokenNotifierCache.Get(mem.Op.GetConfig(params...))
}

func (d *Driver) NewAuditTransaction(_ driver.Config, params ...string) (driver.AuditTransactionDB, error) {
	return d.AuditTxCache.Get(mem.Op.GetConfig(params...))
}

func (d *Driver) NewOwnerTransaction(_ driver.Config, params ...string) (driver.TokenTransactionDB, error) {
	return d.OwnerTxCache.Get(mem.Op.GetConfig(params...))
}
