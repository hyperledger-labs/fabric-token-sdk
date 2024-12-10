/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identitydb

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
)

type identityDBDriver struct{ driver.IdentityDBDriver }

func (d *identityDBDriver) Open(cp driver.ConfigProvider, tmsID token.TMSID) (driver.IdentityDB, error) {
	return d.OpenIdentityDB(cp, tmsID)
}

type walletDBDriver struct{ driver.IdentityDBDriver }

func (d *walletDBDriver) Open(cp driver.ConfigProvider, tmsID token.TMSID) (driver.WalletDB, error) {
	return d.OpenWalletDB(cp, tmsID)
}

type (
	IdentityHolder = db.DriverHolder[driver.IdentityDB, driver.IdentityDB, *identityDBDriver]
	WalletHolder   = db.DriverHolder[driver.WalletDB, driver.WalletDB, *walletDBDriver]
)

var (
	identityHolder = db.NewDriverHolder[driver.IdentityDB, driver.IdentityDB, *identityDBDriver](utils.IdentityFunc[driver.IdentityDB]())
	walletHolder   = db.NewDriverHolder[driver.WalletDB, driver.WalletDB, *walletDBDriver](utils.IdentityFunc[driver.WalletDB]())
)

func Register(name string, driver driver.IdentityDBDriver) {
	identityHolder.Register(name, &identityDBDriver{driver})
	walletHolder.Register(name, &walletDBDriver{driver})
}

func Drivers() []string { return identityHolder.DriverNames() }

type Manager struct {
	identityManager *db.Manager[driver.IdentityDB, driver.IdentityDB, *identityDBDriver]
	walletManager   *db.Manager[driver.WalletDB, driver.WalletDB, *walletDBDriver]
}

func NewManager(drivers []db.NamedDriver[driver.IdentityDBDriver], cp driver.ConfigProvider, config db.Config) *Manager {
	identityDrivers := make([]db.NamedDriver[*identityDBDriver], len(drivers))
	walletDrivers := make([]db.NamedDriver[*walletDBDriver], len(drivers))
	for i, driver := range drivers {
		identityDrivers[i] = db.NamedDriver[*identityDBDriver]{Name: driver.Name, Driver: &identityDBDriver{IdentityDBDriver: driver.Driver}}
		walletDrivers[i] = db.NamedDriver[*walletDBDriver]{Name: driver.Name, Driver: &walletDBDriver{IdentityDBDriver: driver.Driver}}
	}
	identityHolder := db.NewDriverHolder(utils.IdentityFunc[driver.IdentityDB](), identityDrivers...)
	walletHolder := db.NewDriverHolder(utils.IdentityFunc[driver.WalletDB](), walletDrivers...)
	return &Manager{
		identityManager: identityHolder.NewManager(cp, config),
		walletManager:   walletHolder.NewManager(cp, config),
	}
}

func (m *Manager) IdentityDBByTMSId(tmsID token.TMSID) (driver.IdentityDB, error) {
	return m.identityManager.DBByTMSId(tmsID)
}

func (m *Manager) WalletDBByTMSId(tmsID token.TMSID) (driver.WalletDB, error) {
	return m.walletManager.DBByTMSId(tmsID)
}
