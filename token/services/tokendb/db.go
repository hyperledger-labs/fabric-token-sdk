/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tokendb

import (
	"reflect"
	"sort"
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/pkg/errors"
)

var (
	logger    = flogging.MustGetLogger("token-sdk.tokendb")
	driversMu sync.RWMutex
	drivers   = make(map[string]driver.TokenDBDriver)
)

// Register makes a DB driver available by the provided name.
// If Register is called twice with the same name or if driver is nil,
// it panics.
func Register(name string, driver driver.TokenDBDriver) {
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

type TokenRecord = driver.TokenRecord

type Transaction struct {
	driver.TokenDBTransaction
}

// DB is a database that stores token transactions related information
type DB struct {
	driver.TokenDB
}

func (d *DB) NewTransaction() (*Transaction, error) {
	tx, err := d.TokenDB.NewTokenDBTransaction()
	if err != nil {
		return nil, err
	}
	return &Transaction{TokenDBTransaction: tx}, nil
}

func newDB(p driver.TokenDB) *DB {
	return &DB{
		TokenDB: p,
	}
}

type Config interface {
	DriverFor(tmsID token.TMSID) (string, error)
}

// Manager handles the databases
type Manager struct {
	sp     view.ServiceProvider
	config Config

	mutex sync.Mutex
	dbs   map[string]*DB
}

// NewManager creates a new DB manager.
func NewManager(sp view.ServiceProvider, config Config) *Manager {
	return &Manager{
		sp:     sp,
		config: config,
		dbs:    map[string]*DB{},
	}
}

// DBByTMSId returns a DB for the given TMS id
func (m *Manager) DBByTMSId(id token.TMSID) (*DB, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	logger.Debugf("get tokendb for [%s]", id)
	c, ok := m.dbs[id.String()]
	if !ok {
		driverName, err := m.config.DriverFor(id)
		if err != nil {
			return nil, errors.Wrapf(err, "no driver found for [%s]", id)
		}
		d := drivers[driverName]
		if d == nil {
			return nil, errors.Errorf("no driver found for [%s]", driverName)
		}
		driverInstance, err := d.Open(m.sp, id)
		if err != nil {
			return nil, errors.Wrapf(err, "failed instantiating tokendb driver [%s]", driverName)
		}
		c = newDB(driverInstance)
		m.dbs[id.String()] = c
	}
	return c, nil
}

var (
	managerType = reflect.TypeOf((*Manager)(nil))
)

// GetByTMSId returns the DB for the given TMS id.
// Nil might be returned if the wallet is not found or an error occurred.
func GetByTMSId(sp view.ServiceProvider, tmsID token.TMSID) (*DB, error) {
	s, err := sp.GetService(managerType)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get manager service")
	}
	c, err := s.(*Manager).DBByTMSId(tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get db for wallet [%s]", tmsID)
	}
	return c, nil
}
