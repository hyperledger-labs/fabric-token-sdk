/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package auditdb

import (
	"math/big"
	"sort"
	"sync"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditor/auditdb/driver"
	"github.com/pkg/errors"
	"go.uber.org/atomic"
)

var logger = flogging.MustGetLogger("token-sdk.zkat.auditdb")

var (
	driversMu sync.RWMutex
	drivers   = make(map[string]driver.Driver)
)

// Register makes a AuditDB driver available by the provided name.
// If Register is called twice with the same name or if driver is nil,
// it panics.
func Register(name string, driver driver.Driver) {
	driversMu.Lock()
	defer driversMu.Unlock()
	if driver == nil {
		panic("auditor: Register driver is nil")
	}
	if _, dup := drivers[name]; dup {
		panic("auditor: Register called twice for driver " + name)
	}
	drivers[name] = driver
}

func unregisterAllDrivers() {
	driversMu.Lock()
	defer driversMu.Unlock()
	// For tests.
	drivers = make(map[string]driver.Driver)
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

type Status string

const (
	Pending Status = "Pending"
	Valid   Status = "Confirmed"
)

type QueryExecutor struct {
	db     *AuditDB
	closed bool
}

func (qe *QueryExecutor) NewPaymentsFilter() *PaymentsFilter {
	return &PaymentsFilter{
		db: qe.db,
	}
}

func (qe *QueryExecutor) NewHoldingsFilter() *HoldingsFilter {
	return &HoldingsFilter{
		db: qe.db,
	}
}

func (qe *QueryExecutor) Done() {
	if qe.closed {
		return
	}
	qe.db.counter.Dec()
	qe.db.storeLock.RUnlock()
	qe.closed = true
}

type AuditDB struct {
	counter atomic.Int32

	// the vault handles access concurrency to the store using storeLock.
	// In particular:
	// * when a directQueryExecutor is returned, it holds a read-lock;
	//   when Done is called on it, the lock is released.
	// * when an interceptor is returned (using NewRWSet (in case the
	//   transaction context is generated from nothing) or GetRWSet
	//   (in case the transaction context is received from another node)),
	//   it holds a read-lock; when Done is called on it, the lock is released.
	// * an exclusive lock is held when Commit is called.
	db        driver.AuditDB
	storeLock sync.RWMutex

	eIDsLocks sync.Map
}

func newAuditDB(p driver.AuditDB) *AuditDB {
	return &AuditDB{db: p, eIDsLocks: sync.Map{}}
}

func (db *AuditDB) Append(record *token.AuditRecord) error {
	logger.Debugf("Appending new record... [%d]", db.counter)
	db.storeLock.Lock()
	defer db.storeLock.Unlock()
	logger.Debug("lock acquired")

	if err := db.db.BeginUpdate(); err != nil {
		return errors.WithMessagef(err, "begin update for txid '%s' failed", record.TxID)
	}

	inputs := record.Inputs
	outputs := record.Ouputs

	// compute the payment done in the transaction
	eIDs := outputs.EnrollmentIDs()
	tokenTypes := outputs.TokenTypes()
	for _, eID := range eIDs {
		for _, tokenType := range tokenTypes {
			sent := inputs.ByEnrollmentID(eID).ByType(tokenType).Sum().ToBigInt()
			received := outputs.ByEnrollmentID(eID).ByType(tokenType).Sum().ToBigInt()
			diff := sent.Sub(sent, received)
			if diff.Cmp(big.NewInt(0)) <= 0 {
				continue
			}

			if err := db.db.AddRecord(&driver.Record{
				TxID:         record.TxID,
				ActionIndex:  0,
				EnrollmentID: eID,
				Amount:       diff.Neg(diff),
				Type:         tokenType,
				Status:       driver.Pending,
			}); err != nil {
				if err1 := db.db.Discard(); err1 != nil {
					logger.Errorf("got error %s; discarding caused %s", err.Error(), err1.Error())
				}
				return err
			}
		}
	}

	// compute what received in the transaction
	eIDs = outputs.EnrollmentIDs()
	tokenTypes = outputs.TokenTypes()
	for _, eID := range eIDs {
		for _, tokenType := range tokenTypes {
			received := outputs.ByEnrollmentID(eID).ByType(tokenType).Sum().ToBigInt()
			sent := inputs.ByEnrollmentID(eID).ByType(tokenType).Sum().ToBigInt()
			diff := received.Sub(received, sent)
			if diff.Cmp(big.NewInt(0)) <= 0 {
				// Nothing received
				continue
			}

			if err := db.db.AddRecord(&driver.Record{
				TxID:         record.TxID,
				ActionIndex:  0,
				EnrollmentID: eID,
				Amount:       diff,
				Type:         tokenType, Status: driver.Pending,
			}); err != nil {
				if err1 := db.db.Discard(); err1 != nil {
					logger.Errorf("got error %s; discarding caused %s", err.Error(), err1.Error())
				}
				return err
			}
		}
	}

	if err := db.db.Commit(); err != nil {
		return errors.WithMessagef(err, "committing tx for txid '%s' failed", record.TxID)
	}

	logger.Debugf("Appending new completed without errors")
	return nil
}

func (db *AuditDB) NewQueryExecutor() *QueryExecutor {
	db.counter.Inc()
	db.storeLock.RLock()

	return &QueryExecutor{db: db}
}

func (db *AuditDB) SetStatus(txID string, status Status) error {
	logger.Debugf("Set status [%s][%s]...[%d]", txID, status, db.counter)
	db.storeLock.Lock()
	defer db.storeLock.Unlock()
	logger.Debug("lock acquired")

	if err := db.db.BeginUpdate(); err != nil {
		return errors.WithMessagef(err, "begin update for txid '%s' failed", txID)
	}

	if err := db.db.SetStatus(txID, driver.Status(status)); err != nil {
		if err1 := db.db.Discard(); err1 != nil {
			logger.Errorf("got error %s; discarding caused %s", err.Error(), err1.Error())
		}
		return errors.Wrapf(err, "failed setting status [%s][%s]", txID, status)
	}

	if err := db.db.Commit(); err != nil {
		return errors.WithMessagef(err, "committing tx for txid '%s' failed", txID)
	}

	logger.Debugf("Set status [%s][%s]...[%d] done without errors", txID, status, db.counter)
	return nil
}

func (db *AuditDB) AcquireLocks(eIDs ...string) error {
	for _, id := range deduplicate(eIDs) {
		lock, _ := db.eIDsLocks.LoadOrStore(id, &sync.RWMutex{})
		lock.(*sync.RWMutex).Lock()
	}
	return nil
}

func (db *AuditDB) Unlock(eIDs ...string) {
	for _, id := range deduplicate(eIDs) {
		lock, ok := db.eIDsLocks.Load(id)
		if !ok {
			logger.Warnf("unlock for enrollment id [%s] not possible, lock never acquired", id)
			continue
		}
		lock.(*sync.RWMutex).Unlock()
	}
}

type Manager struct {
	sp     view2.ServiceProvider
	driver string
	mutex  sync.Mutex
	dbs    map[string]*AuditDB
}

func NewManager(sp view2.ServiceProvider, driver string) *Manager {
	return &Manager{
		sp:     sp,
		driver: driver,
		dbs:    map[string]*AuditDB{},
	}
}

func (cm *Manager) AuditDB(w *token.AuditorWallet) (*AuditDB, error) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	id := w.ID()
	c, ok := cm.dbs[id]
	if !ok {
		driver, err := drivers[cm.driver].Open(cm.sp, "")
		if err != nil {
			return nil, errors.Wrapf(err, "failed instantiating audit db driver")
		}
		c = newAuditDB(driver)
		cm.dbs[id] = c
	}
	return c, nil
}

func GetAuditDB(sp view2.ServiceProvider, w *token.AuditorWallet) *AuditDB {
	s, err := sp.GetService(&Manager{})
	if err != nil {
		panic(err)
	}
	c, err := s.(*Manager).AuditDB(w)
	if err != nil {
		panic(err)
	}
	return c
}

func deduplicate(source []string) []string {
	support := make(map[string]bool)
	var res []string
	for _, item := range source {
		if _, value := support[item]; !value {
			support[item] = true
			res = append(res, item)
		}
	}
	return res
}
