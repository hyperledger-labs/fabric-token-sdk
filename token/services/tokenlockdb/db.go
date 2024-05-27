/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tokenlockdb

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/core"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/drivers"
)

var holder = drivers.NewDBHolder[*DB, driver.TokenLockDB, driver.TokenLockDBDriver](newDB)

func Register(name string, driver driver.TokenLockDBDriver) { holder.Register(name, driver) }

func Drivers() []string { return holder.DriverNames() }

type DB struct{ driver.TokenLockDB }

func newDB(p driver.TokenLockDB) *DB { return &DB{TokenLockDB: p} }

type Manager = drivers.DBManager[*DB, driver.TokenLockDB, driver.TokenLockDBDriver]

func NewManager(cp core.ConfigProvider, config drivers.Config) *Manager {
	return holder.NewManager(cp, config)
}
