/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identitydb

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/common/utils"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/drivers"
)

type identityDBDriver struct{ driver.IdentityDBDriver }

func (d *identityDBDriver) Open(cp driver.ConfigProvider, tmsID token.TMSID) (driver.IdentityDB, error) {
	return d.OpenIdentityDB(cp, tmsID)
}

type walletDBDriver struct{ driver.IdentityDBDriver }

func (d *walletDBDriver) Open(cp driver.ConfigProvider, tmsID token.TMSID) (driver.WalletDB, error) {
	return d.OpenWalletDB(cp, tmsID)
}

var (
	identityHolder = drivers.NewDBHolder[driver.IdentityDB, driver.IdentityDB, *identityDBDriver](utils.IdentityFunc[driver.IdentityDB]())
	walletHolder   = drivers.NewDBHolder[driver.WalletDB, driver.WalletDB, *walletDBDriver](utils.IdentityFunc[driver.WalletDB]())
)

func Register(name string, driver driver.IdentityDBDriver) {
	identityHolder.Register(name, &identityDBDriver{driver})
	walletHolder.Register(name, &walletDBDriver{driver})
}

func Drivers() []string { return identityHolder.DriverNames() }

type Manager struct {
	identityManager *drivers.DBManager[driver.IdentityDB, driver.IdentityDB, *identityDBDriver]
	walletManager   *drivers.DBManager[driver.WalletDB, driver.WalletDB, *walletDBDriver]
}

func NewManager(cp core.ConfigProvider, config drivers.Config) *Manager {
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
