/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package memory

import (
	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver"
	mem "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/memory"
	driver3 "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/driver"
	sqlite2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/sqlite"
)

type Driver sqlite2.Driver

func NewNamedDriver() driver3.NamedDriver {
	return driver3.NamedDriver{
		Name:   mem.Persistence,
		Driver: NewDriver(),
	}
}

func NewDriver() *Driver {
	return (*Driver)(sqlite2.NewDriver(nil))
}

func (d *Driver) NewTokenLock(_ driver2.PersistenceName, params ...string) (driver3.TokenLockStore, error) {
	return d.TokenLock.Get(mem.Op.GetConfig(params...))
}

func (d *Driver) NewWallet(_ driver2.PersistenceName, params ...string) (driver3.WalletStore, error) {
	return d.Wallet.Get(mem.Op.GetConfig(params...))
}

func (d *Driver) NewIdentity(_ driver2.PersistenceName, params ...string) (driver3.IdentityStore, error) {
	return d.Identity.Get(mem.Op.GetConfig(params...))
}

func (d *Driver) NewKeyStore(_ driver2.PersistenceName, params ...string) (driver3.KeyStore, error) {
	return d.KeyStore.Get(mem.Op.GetConfig(params...))
}

func (d *Driver) NewToken(_ driver2.PersistenceName, params ...string) (driver3.TokenStore, error) {
	return d.Token.Get(mem.Op.GetConfig(params...))
}

func (d *Driver) NewTokenNotifier(_ driver2.PersistenceName, params ...string) (driver3.TokenNotifier, error) {
	return d.TokenNotifier.Get(mem.Op.GetConfig(params...))
}

func (d *Driver) NewAuditTransaction(_ driver2.PersistenceName, params ...string) (driver3.AuditTransactionStore, error) {
	return d.AuditTx.Get(mem.Op.GetConfig(params...))
}

func (d *Driver) NewOwnerTransaction(_ driver2.PersistenceName, params ...string) (driver3.TokenTransactionStore, error) {
	return d.OwnerTx.Get(mem.Op.GetConfig(params...))
}
