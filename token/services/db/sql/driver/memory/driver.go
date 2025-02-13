/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package memory

import (
	mem "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/memory"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	common2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/driver/sql"
	sqlite2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/sqlite"
)

type Driver struct{}

func NewDriver() driver.NamedDriver {
	return driver.NamedDriver{
		Name:   mem.MemoryPersistence,
		Driver: &Driver{},
	}
}

func (d *Driver) NewTokenLock(opts common2.Opts) (driver.TokenLockDB, error) {
	return sql.OpenSqlite(opts, sqlite2.NewTokenLockDB)
}

func (d *Driver) NewWallet(opts common2.Opts) (driver.WalletDB, error) {
	return sql.OpenSqlite(opts, sqlite2.NewWalletDB)
}

func (d *Driver) NewIdentity(opts common2.Opts) (driver.IdentityDB, error) {
	return sql.OpenSqlite(opts, sqlite2.NewIdentityDB)
}

func (d *Driver) NewToken(opts common2.Opts) (driver.TokenDB, error) {
	return sql.OpenSqlite(opts, sqlite2.NewTokenDB)
}

func (d *Driver) NewTokenNotifier(opts common2.Opts) (driver.TokenNotifier, error) {
	return sql.OpenSqlite(opts, sqlite2.NewTokenNotifier)
}

func (d *Driver) NewAuditTransaction(opts common2.Opts) (driver.AuditTransactionDB, error) {
	return sql.OpenSqlite(opts, sqlite2.NewAuditTransactionDB)
}

func (d *Driver) NewOwnerTransaction(opts common2.Opts) (driver.TokenTransactionDB, error) {
	return sql.OpenSqlite(opts, sqlite2.NewTransactionDB)
}
