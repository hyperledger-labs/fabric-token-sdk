/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identitydb

import (
	"sort"
	"sync"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/pkg/errors"
)

var (
	driversMu sync.RWMutex
	drivers   = make(map[string]driver.IdentityDBDriver)
	logger    = logging.MustGetLogger("token-sdk.services.identitydb")
)

// Register makes a DB driver available by the provided name.
// If Register is called twice with the same name or if driver is nil,
// it panics.
func Register(name string, driver driver.IdentityDBDriver) {
	driversMu.Lock()
	defer driversMu.Unlock()
	if driver == nil {
		panic("Register driver is nil")
	}
	if _, dup := drivers[name]; dup {
		panic("Register called twice for driver " + name)
	}
	drivers[name] = driver
}

// Drivers returns a sorted list of the names of the registered drivers.
func Drivers() []string {
	driversMu.RLock()
	defer driversMu.RUnlock()
	list := make([]string, 0, len(drivers))
	for name := range drivers {
		list = append(list, name)
	}
	sort.Strings(list)
	return list
}

type Config interface {
	DriverFor(tmsID token.TMSID) (string, error)
}

type IdType = string

// Manager handles the databases
type Manager struct {
	cp     core.ConfigProvider
	config Config

	mutex       sync.Mutex
	identityDBs map[string]driver.IdentityDB
	walletDBs   map[string]driver.WalletDB
}

// NewManager creates a new DB manager.
func NewManager(cp core.ConfigProvider, config Config) *Manager {
	return &Manager{
		cp:          cp,
		config:      config,
		identityDBs: map[string]driver.IdentityDB{},
		walletDBs:   map[string]driver.WalletDB{},
	}
}

// IdentityDBByTMSId returns a DB for the given TMS id
func (m *Manager) IdentityDBByTMSId(tmsID token.TMSID) (driver.IdentityDB, error) {
	logger.Debugf("get identity db for [%s]", tmsID)
	m.mutex.Lock()
	defer m.mutex.Unlock()
	logger.Debugf("lock acquired for identity db for [%s]", tmsID)

	c, ok := m.identityDBs[tmsID.String()]
	if ok {
		logger.Debugf("identity db for [%s] found, return it", tmsID)
		return c, nil
	}
	logger.Debugf("identity db for [%s] not found, instantiate it", tmsID)
	driverName, err := m.config.DriverFor(tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "no driver found for [%s]", tmsID)
	}
	d := drivers[driverName]
	if d == nil {
		return nil, errors.Errorf("no driver found for [%s]", driverName)
	}
	identityDB, err := d.OpenIdentityDB(m.cp, tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed instantiating identitydb driver [%s] for id [%s]", driverName, tmsID)
	}
	m.identityDBs[tmsID.String()] = identityDB

	return identityDB, nil
}

// WalletDBByTMSId returns a DB for the given TMS id
func (m *Manager) WalletDBByTMSId(tmsID token.TMSID) (driver.WalletDB, error) {
	logger.Debugf("get wallet db for [%s]", tmsID)
	m.mutex.Lock()
	defer m.mutex.Unlock()
	logger.Debugf("lock acquired for wallet db for [%s]", tmsID)

	c, ok := m.walletDBs[tmsID.String()]
	if ok {
		logger.Debugf("wallet db for [%s] found, return it", tmsID)
		return c, nil
	}
	logger.Debugf("wallet db for [%s] not found, instantiate it", tmsID)
	driverName, err := m.config.DriverFor(tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "no driver found for [%s]", tmsID)
	}
	d := drivers[driverName]
	if d == nil {
		return nil, errors.Errorf("no driver found for [%s]", driverName)
	}
	walletDB, err := d.OpenWalletDB(m.cp, tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed instantiating walletdb driver [%s]", driverName)
	}
	m.walletDBs[tmsID.String()] = walletDB

	return walletDB, nil
}
