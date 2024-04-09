/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttxdb

import (
	"math/big"
	"reflect"
	"sort"
	"sync"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db"

	"go.uber.org/zap/zapcore"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/pkg/errors"
	"go.uber.org/atomic"
)

var (
	logger    = flogging.MustGetLogger("token-sdk.ttxdb")
	driversMu sync.RWMutex
	drivers   = make(map[string]driver.TTXDBDriver)
)

// Register makes a DB driver available by the provided name.
// If Register is called twice with the same name or if driver is nil,
// it panics.
func Register(name string, driver driver.TTXDBDriver) {
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

// TxStatus is the status of a transaction
type TxStatus = driver.TxStatus

const (
	// Unknown is the status of a transaction that is unknown
	Unknown = driver.Unknown
	// Pending is the status of a transaction that has been submitted to the ledger
	Pending = driver.Pending
	// Confirmed is the status of a transaction that has been confirmed by the ledger
	Confirmed = driver.Confirmed
	// Deleted is the status of a transaction that has been deleted due to a failure to commit
	Deleted = driver.Deleted
)

// ActionType is the type of action performed by a transaction.
type ActionType = driver.ActionType

const (
	// Issue is the action type for issuing tokens.
	Issue ActionType = iota
	// Transfer is the action type for transferring tokens.
	Transfer
	// Redeem is the action type for redeeming tokens.
	Redeem
)

// TransactionRecord is a more finer-grained version of a movement record.
// Given a Token Transaction, for each token action in the Token Request,
// a transaction record is created for each unique enrollment ID found in the outputs.
// The transaction record contains the total amount of the token type that was transferred to/from that enrollment ID
// in that action.
type TransactionRecord = driver.TransactionRecord

// ValidationRecord is a more finer-grained version of a movement record.
// Given a Token Transaction, for each token action in the Token Request,
// a transaction record is created for each unique enrollment ID found in the outputs.
// The transaction record contains the total amount of the token type that was transferred to/from that enrollment ID
// in that action.
type ValidationRecord = driver.ValidationRecord

// TransactionIterator is an iterator over transaction records
type TransactionIterator struct {
	it driver.TransactionIterator
}

// Close closes the iterator. It must be called when done with the iterator.
func (t *TransactionIterator) Close() {
	t.it.Close()
}

// Next returns the next transaction record, if any.
// It returns nil, nil if there are no more records.
func (t *TransactionIterator) Next() (*TransactionRecord, error) {
	next, err := t.it.Next()
	if err != nil {
		return nil, err
	}
	if next == nil {
		return nil, nil
	}
	return next, nil
}

// ValidationRecordsIterator is an iterator over validation records
type ValidationRecordsIterator struct {
	it driver.ValidationRecordsIterator
}

// Close closes the iterator. It must be called when done with the iterator.
func (t *ValidationRecordsIterator) Close() {
	t.it.Close()
}

// Next returns the next validation record, if any.
// It returns nil, nil if there are no more records.
func (t *ValidationRecordsIterator) Next() (*ValidationRecord, error) {
	next, err := t.it.Next()
	if err != nil {
		return nil, err
	}
	if next == nil {
		return nil, nil
	}
	return next, nil
}

// QueryTransactionsParams defines the parameters for querying movements
type QueryTransactionsParams = driver.QueryTransactionsParams

// QueryValidationRecordsParams defines the parameters for querying movements
type QueryValidationRecordsParams = driver.QueryValidationRecordsParams

// QueryExecutor executors queries against the DB
type QueryExecutor struct {
	db     *DB
	closed bool
}

// Transactions returns an iterators of transaction records filtered by the given params.
func (qe *QueryExecutor) Transactions(params QueryTransactionsParams) (*TransactionIterator, error) {
	it, err := qe.db.db.QueryTransactions(params)
	if err != nil {
		return nil, errors.Errorf("failed to query transactions: %s", err)
	}
	return &TransactionIterator{it: it}, nil
}

// ValidationRecords returns an iterators of validation records filtered by the given params.
func (qe *QueryExecutor) ValidationRecords(params QueryValidationRecordsParams) (*ValidationRecordsIterator, error) {
	it, err := qe.db.db.QueryValidations(params)
	if err != nil {
		return nil, errors.Errorf("failed to query validation records: %s", err)
	}
	return &ValidationRecordsIterator{it: it}, nil
}

// Done closes the query executor. It must be called when the query executor is no longer needed.s
func (qe *QueryExecutor) Done() {
	if qe.closed {
		return
	}
	qe.db.counter.Dec()
	qe.db.storeLock.RUnlock()
	qe.closed = true
}

// Wallet models a wallet
type Wallet interface {
	// ID returns the wallet ID
	ID() string
	// TMS returns the TMS of the wallet
	TMS() *token.ManagementService
}

// DB is a database that stores token transactions related information
type DB struct {
	*db.StatusSupport
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
	db        driver.TokenTransactionDB
	storeLock sync.RWMutex

	eIDsLocks sync.Map

	// status related fields
	pendingTXs []string
}

func newDB(p driver.TokenTransactionDB) *DB {
	return &DB{
		StatusSupport: db.NewStatusSupport(),
		db:            p,
		eIDsLocks:     sync.Map{},
		pendingTXs:    make([]string, 0, 10000),
	}
}

// AppendTransactionRecord appends the transaction records corresponding to the passed token request.
func (d *DB) AppendTransactionRecord(req *token.Request) error {
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("Appending new transaction record... [%s][%d]", req.Anchor, d.counter)
	}
	d.storeLock.Lock()
	defer d.storeLock.Unlock()
	logger.Debug("lock acquired")

	ins, outs, err := req.InputsAndOutputs()
	if err != nil {
		return errors.WithMessagef(err, "failed getting inputs and outputs for request [%s]", req.Anchor)
	}
	record := &token.AuditRecord{
		Anchor:  req.Anchor,
		Inputs:  ins,
		Outputs: outs,
	}
	if err := d.db.BeginUpdate(); err != nil {
		d.rollback(err)
		return errors.WithMessagef(err, "begin update for txid [%s] failed", record.Anchor)
	}
	if err := d.appendTokenRequest(req); err != nil {
		d.rollback(err)
		return errors.WithMessagef(err, "append token request for txid [%s] failed", record.Anchor)
	}
	if err := d.appendTransactions(record); err != nil {
		d.rollback(err)
		return errors.WithMessagef(err, "append transactions for txid [%s] failed", record.Anchor)
	}
	if err := d.db.Commit(); err != nil {
		d.rollback(err)
		return errors.WithMessagef(err, "committing tx for txid [%s] failed", record.Anchor)
	}

	logger.Debugf("Appending transaction record new completed without errors")
	return nil
}

// NewQueryExecutor returns a new query executor
func (d *DB) NewQueryExecutor() *QueryExecutor {
	d.counter.Inc()
	d.storeLock.RLock()

	return &QueryExecutor{db: d}
}

// SetStatus sets the status of the audit records with the passed transaction id to the passed status
func (d *DB) SetStatus(txID string, status TxStatus) error {
	logger.Debugf("Set status [%s][%s]...[%d]", txID, status, d.counter)

	d.storeLock.Lock()
	if err := d.db.SetStatus(txID, status); err != nil {
		d.storeLock.Unlock()
		return errors.Wrapf(err, "failed setting status [%s][%s]", txID, status)
	}
	d.storeLock.Unlock()

	// notify the listeners
	d.Notify(db.StatusEvent{
		TxID:           txID,
		ValidationCode: status,
	})
	logger.Debugf("Set status [%s][%s]...[%d] done without errors", txID, status, d.counter)
	return nil
}

// GetStatus return the status of the given transaction id.
// It returns an error if no transaction with that id is found
func (d *DB) GetStatus(txID string) (TxStatus, error) {
	logger.Debugf("Get status [%s]...[%d]", txID, d.counter)
	d.storeLock.Lock()
	defer d.storeLock.Unlock()
	logger.Debug("lock acquired")

	status, err := d.db.GetStatus(txID)
	if err != nil {
		return Unknown, errors.Wrapf(err, "failed geting status [%s]", txID)
	}
	logger.Debugf("Get status [%s][%s]...[%d] done without errors", txID, status, d.counter)
	return status, nil
}

// GetTokenRequest returns the token request bound to the passed transaction id, if available.
func (d *DB) GetTokenRequest(txID string) ([]byte, error) {
	return d.db.GetTokenRequest(txID)
}

// AddTransactionEndorsementAck records the signature of a given endorser for a given transaction
func (d *DB) AddTransactionEndorsementAck(txID string, id view2.Identity, sigma []byte) error {
	return d.db.AddTransactionEndorsementAck(txID, id, sigma)
}

// GetTransactionEndorsementAcks returns the endorsement signatures for the given transaction id
func (d *DB) GetTransactionEndorsementAcks(txID string) (map[string][]byte, error) {
	return d.db.GetTransactionEndorsementAcks(txID)
}

// AppendValidationRecord appends the given validation metadata related to the given token request and transaction id
func (d *DB) AppendValidationRecord(txID string, tr []byte, meta map[string][]byte) error {
	logger.Debugf("Appending new validation record... [%d]", d.counter)
	d.storeLock.Lock()
	defer d.storeLock.Unlock()
	logger.Debug("lock acquired")

	if err := d.db.BeginUpdate(); err != nil {
		d.rollback(err)
		return errors.WithMessagef(err, "begin update for txid [%s] failed", txID)
	}
	if err := d.db.AddValidationRecord(txID, tr, meta); err != nil {
		d.rollback(err)
		return errors.WithMessagef(err, "append validation record for txid [%s] failed", txID)
	}
	if err := d.db.Commit(); err != nil {
		d.rollback(err)
		return errors.WithMessagef(err, "committing tx for txid [%s] failed", txID)
	}

	logger.Debugf("Appending validation record completed without errors")
	return nil
}

func (d *DB) appendTransactions(record *token.AuditRecord) error {
	inputs := record.Inputs
	outputs := record.Outputs

	actionIndex := 0
	timestamp := time.Now()
	for {
		// collect inputs and outputs from the same action
		ins := inputs.Filter(func(t *token.Input) bool {
			return t.ActionIndex == actionIndex
		})
		ous := outputs.Filter(func(t *token.Output) bool {
			return t.ActionIndex == actionIndex
		})
		if ins.Count() == 0 && ous.Count() == 0 {
			logger.Debugf("no actions left for tx [%s][%d]", record.Anchor, actionIndex)
			// no more actions
			break
		}

		// create a transaction record from ins and ous

		// All ins should be for same EID, check this
		inEIDs := ins.EnrollmentIDs()
		if len(inEIDs) > 1 {
			return errors.Errorf("expected at most 1 input enrollment id, got %d, [%v]", len(inEIDs), inEIDs)
		}
		inEID := ""
		if len(inEIDs) == 1 {
			inEID = inEIDs[0]
		}

		outEIDs := ous.EnrollmentIDs()
		outEIDs = append(outEIDs, "")
		outTT := ous.TokenTypes()
		for _, outEID := range outEIDs {
			for _, tokenType := range outTT {
				received := ous.ByEnrollmentID(outEID).ByType(tokenType).Sum()
				if received.Cmp(big.NewInt(0)) <= 0 {
					continue
				}

				tt := driver.Issue
				if len(inEIDs) != 0 {
					if len(outEID) == 0 {
						tt = driver.Redeem
					} else {
						tt = driver.Transfer
					}
				}

				if err := d.db.AddTransaction(&driver.TransactionRecord{
					TxID:         record.Anchor,
					SenderEID:    inEID,
					RecipientEID: outEID,
					TokenType:    tokenType,
					Amount:       received,
					Status:       driver.Pending,
					ActionType:   tt,
					Timestamp:    timestamp,
				}); err != nil {
					if err1 := d.db.Discard(); err1 != nil {
						logger.Errorf("got error [%s]; discarding caused [%s]", err.Error(), err1.Error())
					}
					return err
				}
			}
		}

		actionIndex++
	}
	logger.Debugf("finished appending transactions for tx [%s]", record.Anchor)

	return nil
}

func (d *DB) appendTokenRequest(request *token.Request) error {
	raw, err := request.Bytes()
	if err != nil {
		return errors.Wrapf(err, "failed to marshal token request [%s]", request.Anchor)
	}
	if err := d.db.AddTokenRequest(request.Anchor, raw); err != nil {
		if err1 := d.db.Discard(); err1 != nil {
			logger.Errorf("got error [%s]; discarding caused [%s]", err.Error(), err1.Error())
		}
		return err
	}
	return nil
}

func (d *DB) rollback(err error) {
	if err1 := d.db.Discard(); err1 != nil {
		logger.Errorf("got error %s; discarding caused %s", err.Error(), err1.Error())
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

	logger.Debugf("get ttxdb for [%s]", id)
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
			return nil, errors.Wrapf(err, "failed instantiating ttxdb driver [%s]", driverName)
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
