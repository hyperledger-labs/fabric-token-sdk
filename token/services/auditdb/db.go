/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package auditdb

import (
	"sync"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/drivers"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	"github.com/pkg/errors"
)

var (
	holder = drivers.NewDBHolder[*DB, driver.AuditTransactionDB, driver.AuditDBDriver](newDB)
	logger = logging.MustGetLogger("token-sdk.auditdb")
)

func Register(name string, driver driver.AuditDBDriver) { holder.Register(name, driver) }

func Drivers() []string { return holder.DriverNames() }

type Manager = drivers.DBManager[*DB, driver.AuditTransactionDB, driver.AuditDBDriver]

func NewManager(cp core.ConfigProvider, config drivers.Config) *Manager {
	return holder.NewManager(cp, config)
}

func GetByTMSId(sp token.ServiceProvider, tmsID token.TMSID) (*DB, error) {
	return holder.GetByTMSId(sp, tmsID)
}

func GetProvider(sp token.ServiceProvider) (*Manager, error) {
	return holder.GetProvider(sp)
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

// TxStatusMessage maps TxStatus to string
var TxStatusMessage = driver.TxStatusMessage

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

// MovementRecord is a record of a movement of assets.
// Given a Token Transaction, a movement record is created for each enrollment ID that participated in the transaction
// and each token type that was transferred.
// The movement record contains the total amount of the token type that was transferred to/from the enrollment ID
// in a given token transaction.
type MovementRecord = driver.MovementRecord

// TransactionRecord is a more finer-grained version of a movement record.
// Given a Token Transaction, for each token action in the Token Request,
// a transaction record is created for each unique enrollment ID found in the outputs.
// The transaction record contains the total amount of the token type that was transferred to/from that enrollment ID
// in that action.
type TransactionRecord = driver.TransactionRecord

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

// QueryTransactionsParams defines the parameters for querying movements
type QueryTransactionsParams = driver.QueryTransactionsParams

// QueryTokenRequestsParams defines the parameters for querying token requests
type QueryTokenRequestsParams = driver.QueryTokenRequestsParams

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
	db        driver.AuditTransactionDB
	eIDsLocks sync.Map

	// status related fields
	pendingTXs []string
}

func newDB(p driver.AuditTransactionDB) *DB {
	return &DB{
		StatusSupport: db.NewStatusSupport(),
		db:            p,
		eIDsLocks:     sync.Map{},
		pendingTXs:    make([]string, 0, 10000),
	}
}

// Append appends send and receive movements, and transaction records corresponding to the passed token request
func (d *DB) Append(req *token.Request) error {
	logger.Debugf("appending new record... [%s]", req.Anchor)

	record, err := req.AuditRecord()
	if err != nil {
		return errors.WithMessagef(err, "failed getting audit records for request [%s]", req.Anchor)
	}

	logger.Debugf("parsing new audit record... [%d] in, [%d] out", record.Inputs.Count(), record.Outputs.Count())
	now := time.Now().UTC()
	raw, err := req.Bytes()
	if err != nil {
		return errors.Wrapf(err, "failed to marshal token request [%s]", req.Anchor)
	}
	mov, err := ttxdb.Movements(record, now)
	if err != nil {
		return errors.WithMessage(err, "failed parsing movements from audit record")
	}
	txs, err := ttxdb.TransactionRecords(record, now)
	if err != nil {
		return errors.WithMessage(err, "failed parsing transactions from audit record")
	}

	logger.Debugf("storing new records... [%d,%d,%d]", len(raw), len(mov), len(txs))
	w, err := d.db.BeginAtomicWrite()
	if err != nil {
		return errors.WithMessagef(err, "begin update for txid [%s] failed", record.Anchor)
	}
	if err := w.AddTokenRequest(record.Anchor, raw, req.Metadata.Application); err != nil {
		w.Rollback()
		return errors.WithMessagef(err, "append token request for txid [%s] failed", record.Anchor)
	}
	for _, mv := range mov {
		if err := w.AddMovement(&mv); err != nil {
			w.Rollback()
			return errors.WithMessagef(err, "append sent movements for txid [%s] failed", record.Anchor)
		}
	}
	for _, tx := range txs {
		if err := w.AddTransaction(&tx); err != nil {
			w.Rollback()
			return errors.WithMessagef(err, "append transactions for txid [%s] failed", record.Anchor)
		}
	}
	if err := w.Commit(); err != nil {
		return errors.WithMessagef(err, "committing tx for txid [%s] failed", record.Anchor)
	}

	logger.Debugf("appending new records completed without errors")
	return nil
}

// Transactions returns an iterators of transaction records filtered by the given params.
func (d *DB) Transactions(params QueryTransactionsParams) (driver.TransactionIterator, error) {
	return d.db.QueryTransactions(params)
}

// TokenRequests returns an iterator over the token requests matching the passed params
func (d *DB) TokenRequests(params QueryTokenRequestsParams) (driver.TokenRequestIterator, error) {
	return d.db.QueryTokenRequests(params)
}

// NewPaymentsFilter returns a programmable filter over the payments sent or received by enrollment IDs.
func (d *DB) NewPaymentsFilter() *PaymentsFilter {
	return &PaymentsFilter{
		db: d,
	}
}

// NewHoldingsFilter returns a programmable filter over the holdings owned by enrollment IDs.
func (d *DB) NewHoldingsFilter() *HoldingsFilter {
	return &HoldingsFilter{
		db: d,
	}
}

// SetStatus sets the status of the audit records with the passed transaction id to the passed status
func (d *DB) SetStatus(txID string, status TxStatus, message string) error {
	logger.Debugf("set status [%s][%s]...", txID, status)
	if err := d.db.SetStatus(txID, status, message); err != nil {
		return errors.Wrapf(err, "failed setting status [%s][%s]", txID, driver.TxStatusMessage[status])
	}

	// notify the listeners
	d.Notify(db.StatusEvent{
		TxID:           txID,
		ValidationCode: status,
	})
	logger.Debugf("set status [%s][%s]...done without errors", txID, driver.TxStatusMessage[status])
	return nil
}

// GetStatus return the status of the given transaction id.
// It returns an error if no transaction with that id is found
func (d *DB) GetStatus(txID string) (TxStatus, string, error) {
	logger.Debugf("get status [%s]...", txID)
	status, message, err := d.db.GetStatus(txID)
	if err != nil {
		return Unknown, "", errors.Wrapf(err, "failed geting status [%s]", txID)
	}
	logger.Debugf("Got status [%s][%s]", txID, status)
	return status, message, nil
}

// GetTokenRequest returns the token request bound to the passed transaction id, if available.
func (d *DB) GetTokenRequest(txID string) ([]byte, error) {
	return d.db.GetTokenRequest(txID)
}

// AcquireLocks acquires locks for the passed anchor and enrollment ids.
// This can be used to prevent concurrent read/write access to the audit records of the passed enrollment ids.
func (d *DB) AcquireLocks(anchor string, eIDs ...string) error {
	dedup := deduplicate(eIDs)
	logger.Debugf("Acquire locks for [%s:%v] enrollment ids", anchor, dedup)
	d.eIDsLocks.LoadOrStore(anchor, dedup)
	for _, id := range dedup {
		lock, _ := d.eIDsLocks.LoadOrStore(id, &sync.RWMutex{})
		lock.(*sync.RWMutex).Lock()
		logger.Debugf("Acquire locks for [%s:%v] enrollment id done", anchor, id)
	}
	logger.Debugf("Acquire locks for [%s:%v] enrollment ids...done", anchor, dedup)
	return nil
}

// ReleaseLocks releases the locks associated to the passed anchor
func (d *DB) ReleaseLocks(anchor string) {
	dedupBoxed, ok := d.eIDsLocks.LoadAndDelete(anchor)
	if !ok {
		logger.Debugf("nothing to release for [%s] ", anchor)
		return
	}
	dedup := dedupBoxed.([]string)
	logger.Debugf("Release locks for [%s:%v] enrollment ids", anchor, dedup)
	for _, id := range dedup {
		lock, ok := d.eIDsLocks.Load(id)
		if !ok {
			logger.Warnf("unlock for enrollment id [%d:%s] not possible, lock never acquired", anchor, id)
			continue
		}
		logger.Debugf("unlock lock for [%s:%v] enrollment id done", anchor, id)
		lock.(*sync.RWMutex).Unlock()
	}
	logger.Debugf("Release locks for [%s:%v] enrollment ids...done", anchor, dedup)

}

// deduplicate removes duplicate entries from a slice
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
