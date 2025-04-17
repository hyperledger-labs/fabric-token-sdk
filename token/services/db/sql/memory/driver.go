/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package memory

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/common"
	mem "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/memory"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/sqlite"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	sqlite2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/sqlite"
)

type Driver struct{}

func NewDriver() driver.NamedDriver {
	return driver.NamedDriver{
		Name:   mem.Persistence,
		Driver: &Driver{},
	}
}

func (d *Driver) NewTokenLock(_ driver.Config, params ...string) (driver.TokenLockDB, error) {
	return newPersistenceWithOpts(sqlite2.NewTokenLockDB, params...)
}

func (d *Driver) NewWallet(_ driver.Config, params ...string) (driver.WalletDB, error) {
	return newPersistenceWithOpts(sqlite2.NewWalletDB, params...)
}

func (d *Driver) NewIdentity(_ driver.Config, params ...string) (driver.IdentityDB, error) {
	return newPersistenceWithOpts(sqlite2.NewIdentityDB, params...)
}

func (d *Driver) NewToken(_ driver.Config, params ...string) (driver.TokenDB, error) {
	return newPersistenceWithOpts(sqlite2.NewTokenDB, params...)
}

func (d *Driver) NewTokenNotifier(_ driver.Config, params ...string) (driver.TokenNotifier, error) {
	return newPersistenceWithOpts(sqlite2.NewTokenNotifier, params...)
}

func (d *Driver) NewAuditTransaction(_ driver.Config, params ...string) (driver.AuditTransactionDB, error) {
	return newPersistenceWithOpts(sqlite2.NewAuditTransactionDB, params...)
}

func (d *Driver) NewOwnerTransaction(_ driver.Config, params ...string) (driver.TokenTransactionDB, error) {
	return newPersistenceWithOpts(sqlite2.NewTransactionDB, params...)
}

func newPersistenceWithOpts[V common.DBObject](constructor common.PersistenceConstructor[sqlite.Opts, V], params ...string) (V, error) {
	p, err := constructor(mem.Op.GetOpts(params...))
	if err != nil {
		return utils.Zero[V](), err
	}
	if err := p.CreateSchema(); err != nil {
		return utils.Zero[V](), err
	}

	return p, nil
}
