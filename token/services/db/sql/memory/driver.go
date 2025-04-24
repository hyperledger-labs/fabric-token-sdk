/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package memory

import (
	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver"
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
	return (*Driver)(sqlite2.NewDriver(nil))
}

func (d *Driver) NewTokenLock(_ driver2.PersistenceName, params ...string) (driver.TokenLockStore, error) {
	return d.TokenLock.Get(mem.Op.GetConfig(params...))
}

func (d *Driver) NewWallet(_ driver2.PersistenceName, params ...string) (driver.WalletStore, error) {
	return d.Wallet.Get(mem.Op.GetConfig(params...))
}

func (d *Driver) NewIdentity(_ driver2.PersistenceName, params ...string) (driver.IdentityStore, error) {
	return d.Identity.Get(mem.Op.GetConfig(params...))
}

func (d *Driver) NewToken(_ driver2.PersistenceName, params ...string) (driver.TokenStore, error) {
	return d.Token.Get(mem.Op.GetConfig(params...))
}

func (d *Driver) NewTokenNotifier(_ driver2.PersistenceName, params ...string) (driver.TokenNotifier, error) {
	return d.TokenNotifier.Get(mem.Op.GetConfig(params...))
}

func (d *Driver) NewAuditTransaction(_ driver2.PersistenceName, params ...string) (driver.AuditTransactionStore, error) {
	return d.AuditTx.Get(mem.Op.GetConfig(params...))
}

func (d *Driver) NewOwnerTransaction(_ driver2.PersistenceName, params ...string) (driver.TokenTransactionStore, error) {
	return d.OwnerTx.Get(mem.Op.GetConfig(params...))
}
