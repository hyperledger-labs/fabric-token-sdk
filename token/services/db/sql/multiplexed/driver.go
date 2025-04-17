/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package multiplexed

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/multiplexed"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/pkg/errors"
)

var _ driver.Driver = &Driver{}

type Driver []driver.NamedDriver

func (d *Driver) NewTokenLock(cfg driver.Config, params ...string) (driver.TokenLockDB, error) {
	dr, err := d.getDriver(cfg)
	if err != nil {
		return nil, err
	}
	return dr.NewTokenLock(cfg, params...)
}

func (d *Driver) NewWallet(cfg driver.Config, params ...string) (driver.WalletDB, error) {
	dr, err := d.getDriver(cfg)
	if err != nil {
		return nil, err
	}
	return dr.NewWallet(cfg, params...)
}

func (d *Driver) NewIdentity(cfg driver.Config, params ...string) (driver.IdentityDB, error) {
	dr, err := d.getDriver(cfg)
	if err != nil {
		return nil, err
	}
	return dr.NewIdentity(cfg, params...)
}

func (d *Driver) NewToken(cfg driver.Config, params ...string) (driver.TokenDB, error) {
	dr, err := d.getDriver(cfg)
	if err != nil {
		return nil, err
	}
	return dr.NewToken(cfg, params...)
}

func (d *Driver) NewTokenNotifier(cfg driver.Config, params ...string) (driver.TokenNotifier, error) {
	dr, err := d.getDriver(cfg)
	if err != nil {
		return nil, err
	}
	return dr.NewTokenNotifier(cfg, params...)
}

func (d *Driver) NewAuditTransaction(cfg driver.Config, params ...string) (driver.AuditTransactionDB, error) {
	dr, err := d.getDriver(cfg)
	if err != nil {
		return nil, err
	}
	return dr.NewAuditTransaction(cfg, params...)
}

func (d *Driver) NewOwnerTransaction(cfg driver.Config, params ...string) (driver.TokenTransactionDB, error) {
	dr, err := d.getDriver(cfg)
	if err != nil {
		return nil, err
	}
	return dr.NewOwnerTransaction(cfg, params...)
}

func (d Driver) getDriver(c driver.Config) (driver.Driver, error) {
	t, err := multiplexed.GetDriverType(c)
	if err != nil {
		return nil, err
	}
	for _, dr := range d {
		if dr.Name == t {
			return dr.Driver, nil
		}
	}
	return nil, errors.Errorf("driver %s not found", t)
}
