/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package multiplexed

import (
	driver3 "github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/common"
	driver4 "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/driver"
	"github.com/pkg/errors"
)

var _ driver4.Driver = &Driver{}

func NewDriver(config driver4.Config, ds ...driver4.NamedDriver) Driver {
	drivers := make(map[driver3.PersistenceType]driver4.Driver, len(ds))
	for _, d := range ds {
		drivers[d.Name] = d.Driver
	}
	return Driver{
		drivers: drivers,
		config:  common.NewConfig(config),
	}
}

type Driver struct {
	drivers map[driver3.PersistenceType]driver4.Driver
	config  driver2.PersistenceConfig
}

func (d Driver) NewTokenLock(name driver2.PersistenceName, params ...string) (driver4.TokenLockStore, error) {
	dr, err := d.getDriver(name)
	if err != nil {
		return nil, err
	}
	return dr.NewTokenLock(name, params...)
}

func (d Driver) NewWallet(name driver2.PersistenceName, params ...string) (driver4.WalletStore, error) {
	dr, err := d.getDriver(name)
	if err != nil {
		return nil, err
	}
	return dr.NewWallet(name, params...)
}

func (d Driver) NewIdentity(name driver2.PersistenceName, params ...string) (driver4.IdentityStore, error) {
	dr, err := d.getDriver(name)
	if err != nil {
		return nil, err
	}
	return dr.NewIdentity(name, params...)
}

func (d Driver) NewToken(name driver2.PersistenceName, params ...string) (driver4.TokenStore, error) {
	dr, err := d.getDriver(name)
	if err != nil {
		return nil, err
	}
	return dr.NewToken(name, params...)
}

func (d Driver) NewTokenNotifier(name driver2.PersistenceName, params ...string) (driver4.TokenNotifier, error) {
	dr, err := d.getDriver(name)
	if err != nil {
		return nil, err
	}
	return dr.NewTokenNotifier(name, params...)
}

func (d Driver) NewAuditTransaction(name driver2.PersistenceName, params ...string) (driver4.AuditTransactionStore, error) {
	dr, err := d.getDriver(name)
	if err != nil {
		return nil, err
	}
	return dr.NewAuditTransaction(name, params...)
}

func (d Driver) NewOwnerTransaction(name driver2.PersistenceName, params ...string) (driver4.TokenTransactionStore, error) {
	dr, err := d.getDriver(name)
	if err != nil {
		return nil, err
	}
	return dr.NewOwnerTransaction(name, params...)
}

func (d Driver) getDriver(name driver2.PersistenceName) (driver4.Driver, error) {
	t, err := d.config.GetDriverType(name)
	if err != nil {
		return nil, err
	}
	if dr, ok := d.drivers[t]; ok {
		return dr, nil
	}
	return nil, errors.Errorf("driver %s not found [%s]", t, name)
}
