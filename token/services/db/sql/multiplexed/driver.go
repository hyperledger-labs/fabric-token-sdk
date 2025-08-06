/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package multiplexed

import (
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	driver3 "github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
)

var _ driver.Driver = &Driver{}

func NewDriver(config driver.Config, ds ...driver.NamedDriver) Driver {
	drivers := make(map[driver3.PersistenceType]driver.Driver, len(ds))
	for _, d := range ds {
		drivers[d.Name] = d.Driver
	}
	return Driver{
		drivers: drivers,
		config:  common.NewConfig(config),
	}
}

type Driver struct {
	drivers map[driver3.PersistenceType]driver.Driver
	config  driver2.PersistenceConfig
}

func (d Driver) NewTokenLock(name driver2.PersistenceName, params ...string) (driver.TokenLockStore, error) {
	dr, err := d.getDriver(name)
	if err != nil {
		return nil, err
	}
	return dr.NewTokenLock(name, params...)
}

func (d Driver) NewWallet(name driver2.PersistenceName, params ...string) (driver.WalletStore, error) {
	dr, err := d.getDriver(name)
	if err != nil {
		return nil, err
	}
	return dr.NewWallet(name, params...)
}

func (d Driver) NewIdentity(name driver2.PersistenceName, params ...string) (driver.IdentityStore, error) {
	dr, err := d.getDriver(name)
	if err != nil {
		return nil, err
	}
	return dr.NewIdentity(name, params...)
}

func (d Driver) NewKeyStore(name driver2.PersistenceName, params ...string) (driver.KeyStore, error) {
	dr, err := d.getDriver(name)
	if err != nil {
		return nil, err
	}
	return dr.NewKeyStore(name, params...)
}

func (d Driver) NewToken(name driver2.PersistenceName, params ...string) (driver.TokenStore, error) {
	dr, err := d.getDriver(name)
	if err != nil {
		return nil, err
	}
	return dr.NewToken(name, params...)
}

func (d Driver) NewTokenNotifier(name driver2.PersistenceName, params ...string) (driver.TokenNotifier, error) {
	dr, err := d.getDriver(name)
	if err != nil {
		return nil, err
	}
	return dr.NewTokenNotifier(name, params...)
}

func (d Driver) NewAuditTransaction(name driver2.PersistenceName, params ...string) (driver.AuditTransactionStore, error) {
	dr, err := d.getDriver(name)
	if err != nil {
		return nil, err
	}
	return dr.NewAuditTransaction(name, params...)
}

func (d Driver) NewOwnerTransaction(name driver2.PersistenceName, params ...string) (driver.TokenTransactionStore, error) {
	dr, err := d.getDriver(name)
	if err != nil {
		return nil, err
	}
	return dr.NewOwnerTransaction(name, params...)
}

func (d Driver) getDriver(name driver2.PersistenceName) (driver.Driver, error) {
	t, err := d.config.GetDriverType(name)
	if err != nil {
		return nil, err
	}
	if dr, ok := d.drivers[t]; ok {
		return dr, nil
	}
	return nil, errors.Errorf("driver %s not found [%s]", t, name)
}
