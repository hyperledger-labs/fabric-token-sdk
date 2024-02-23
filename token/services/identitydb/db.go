/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identitydb

import (
	"sort"
	"sync"

	"github.com/IBM/idemix/bccsp/keystore"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/kvs"
	"github.com/pkg/errors"
)

var (
	driversMu sync.RWMutex
	drivers   = make(map[string]driver.IdentityDBDriver)
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

// Manager handles the databases
type Manager struct {
	sp     view.ServiceProvider
	config Config

	mutex       sync.Mutex
	identityDBs map[string]driver.IdentityDB
	walletDBs   map[string]driver.WalletDB
}

// NewManager creates a new DB manager.
func NewManager(sp view.ServiceProvider, config Config) *Manager {
	return &Manager{
		sp:          sp,
		config:      config,
		identityDBs: map[string]driver.IdentityDB{},
		walletDBs:   map[string]driver.WalletDB{},
	}
}

// IdentityDBByTMSId returns a DB for the given TMS id
func (m *Manager) IdentityDBByTMSId(tmsID token.TMSID, id string) (driver.IdentityDB, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	c, ok := m.identityDBs[tmsID.String()+"_"+id]
	if ok {
		return c, nil
	}
	driverName, err := m.config.DriverFor(tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "no driver found for [%s]", tmsID)
	}
	d := drivers[driverName]
	if d == nil {
		return nil, errors.Errorf("no driver found for [%s]", driverName)
	}
	identityDB, err := d.OpenIdentityDB(m.sp, tmsID, id)
	if err != nil {
		return nil, errors.Wrapf(err, "failed instantiating identitydb driver [%s] for id [%s]", driverName, id)
	}
	m.identityDBs[tmsID.String()+"_"+id] = identityDB

	return identityDB, nil
}

// WalletDBByTMSId returns a DB for the given TMS id
func (m *Manager) WalletDBByTMSId(tmsID token.TMSID) (driver.WalletDB, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	c, ok := m.walletDBs[tmsID.String()]
	if ok {
		return c, nil
	}
	driverName, err := m.config.DriverFor(tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "no driver found for [%s]", tmsID)
	}
	d := drivers[driverName]
	if d == nil {
		return nil, errors.Errorf("no driver found for [%s]", driverName)
	}
	walletDB, err := d.OpenWalletDB(m.sp, tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed instantiating walletdb driver [%s]", driverName)
	}
	m.walletDBs[tmsID.String()] = walletDB

	return walletDB, nil
}

type DBStorageProvider struct {
	kvs     kvs.KVS
	manager *Manager
}

type IdType = string

var idMap = map[IdType]string{
	"idemix": "i",
	"x509":   "x",
}

func NewDBStorageProvider(kvs kvs.KVS, manager *Manager) *DBStorageProvider {
	return &DBStorageProvider{kvs: kvs, manager: manager}
}

func (s *DBStorageProvider) OpenWalletDB(tmsID token.TMSID) (driver.WalletDB, error) {
	return s.manager.WalletDBByTMSId(tmsID)
}

func (s *DBStorageProvider) OpenIdentityDB(tmsID token.TMSID, id IdType) (driver.IdentityDB, error) {
	if mapped, ok := idMap[id]; ok {
		id = mapped // Tables are not allowed to have numbers in their names, so x509 is invalid
	}
	return s.manager.IdentityDBByTMSId(tmsID, id)
}

func (s *DBStorageProvider) NewKeystore() (keystore.KVS, error) {
	return s.kvs, nil
}
