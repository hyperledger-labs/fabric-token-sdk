/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package auditdb

import (
	"context"
	"reflect"
	"slices"
	"sync"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	cdriver "github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/common"
	dbdriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/multiplexed"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/ttxdb"
)

var (
	managerType = reflect.TypeOf((*StoreServiceManager)(nil))
	logger      = logging.MustGetLogger()
)

type tokenRequest interface {
	ID() token.RequestAnchor
	AuditRecord(ctx context.Context) (*token.AuditRecord, error)
	Bytes() ([]byte, error)
	AllApplicationMetadata() map[string][]byte
	PublicParamsHash() token.PPHash
	String() string
}

//go:generate counterfeiter -o mock/audit_store_service_manager.go --fake-name AuditStoreServiceManager . StoreServiceManager
type StoreServiceManager db.StoreServiceManager[*StoreService]

func NewStoreServiceManager(cp db.ConfigService, drivers multiplexed.Driver) StoreServiceManager {
	return db.NewStoreServiceManager(cp, "auditdb.persistence", drivers.NewAuditTransaction, NewStoreService)
}

func GetByTMSID(sp token.ServiceProvider, tmsID token.TMSID) (*StoreService, error) {
	s, err := sp.GetService(managerType)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get manager service")
	}
	c, err := s.(StoreServiceManager).StoreServiceByTMSId(tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get db for tms [%s]", tmsID)
	}

	return c, nil
}

// TxStatus is the status of a transaction
type TxStatus = dbdriver.TxStatus

const (
	// Unknown is the status of a transaction that is unknown
	Unknown = dbdriver.Unknown
	// Pending is the status of a transaction that has been submitted to the ledger
	Pending = dbdriver.Pending
	// Confirmed is the status of a transaction that has been confirmed by the ledger
	Confirmed = dbdriver.Confirmed
	// Deleted is the status of a transaction that has been deleted due to a failure to commit
	Deleted = dbdriver.Deleted
)

// TxStatusMessage maps TxStatus to string
var TxStatusMessage = dbdriver.TxStatusMessage

// ActionType is the type of action performed by a transaction.
type ActionType = dbdriver.ActionType

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
type MovementRecord = dbdriver.MovementRecord

// TransactionRecord is a more finer-grained version of a movement record.
// Given a Token Transaction, for each token action in the Token Request,
// a transaction record is created for each unique enrollment ID found in the outputs.
// The transaction record contains the total amount of the token type that was transferred to/from that enrollment ID
// in that action.
type TransactionRecord = dbdriver.TransactionRecord

// QueryTransactionsParams defines the parameters for querying movements
type QueryTransactionsParams = dbdriver.QueryTransactionsParams

// QueryTokenRequestsParams defines the parameters for querying token requests
type QueryTokenRequestsParams = dbdriver.QueryTokenRequestsParams

// Pagination defines the pagination for querying movements
type Pagination = cdriver.Pagination

// PageTransactionsIterator iterator defines the pagination iterator for movements query results
type PageTransactionsIterator = cdriver.PageIterator[*TransactionRecord]

// Wallet models a wallet
type Wallet interface {
	// ID returns the wallet ID
	ID() string
	// TMS returns the TMS of the wallet
	TMS() *token.ManagementService
}

// StoreService is a database that stores token transactions related information
type StoreService struct {
	*common.StatusSupport
	db        dbdriver.AuditTransactionStore
	eIDsLocks sync.Map

	// status related fields
	pendingTXs []string
}

func NewStoreService(p dbdriver.AuditTransactionStore) (*StoreService, error) {
	return &StoreService{
		StatusSupport: common.NewStatusSupport(),
		db:            p,
		eIDsLocks:     sync.Map{},
		pendingTXs:    make([]string, 0, 10000),
	}, nil
}

func (d *StoreService) NewTransaction() (dbdriver.TransactionStoreTransaction, error) {
	return d.db.NewTransactionStoreTransaction()
}

// Append appends send and receive movements, and transaction records corresponding to the passed token request
func (d *StoreService) Append(ctx context.Context, req tokenRequest) error {
	logger.DebugfContext(ctx, "appending new record... [%s]", req)

	record, err := req.AuditRecord(ctx)
	if err != nil {
		return errors.WithMessagef(err, "failed getting audit records for request [%s]", req)
	}

	logger.DebugfContext(ctx, "parsing new audit record... [%d] in, [%d] out", record.Inputs.Count(), record.Outputs.Count())
	now := time.Now().UTC()
	raw, err := req.Bytes()
	if err != nil {
		return errors.Wrapf(err, "failed to marshal token request [%s]", req)
	}
	mov, err := ttxdb.Movements(ctx, record, now)
	if err != nil {
		return errors.WithMessagef(err, "failed parsing movements from audit record")
	}
	txs, err := ttxdb.TransactionRecords(ctx, record, now)
	if err != nil {
		return errors.WithMessagef(err, "failed parsing transactions from audit record")
	}

	logger.DebugfContext(ctx, "storing new records... [%d,%d,%d]", len(raw), len(mov), len(txs))
	w, err := d.db.NewTransactionStoreTransaction()
	if err != nil {
		return errors.WithMessagef(err, "begin update for txid [%s] failed", record.Anchor)
	}
	if err := w.AddTokenRequest(
		ctx,
		string(record.Anchor),
		raw,
		req.AllApplicationMetadata(),
		record.Attributes,
		req.PublicParamsHash(),
	); err != nil {
		w.Rollback()

		return errors.WithMessagef(err, "append token request for txid [%s] failed", record.Anchor)
	}
	if err := w.AddMovement(ctx, mov...); err != nil {
		w.Rollback()

		return errors.WithMessagef(err, "append sent movements for txid [%s] failed", record.Anchor)
	}

	if err := w.AddTransaction(ctx, txs...); err != nil {
		w.Rollback()

		return errors.WithMessagef(err, "append transactions for txid [%s] failed", record.Anchor)
	}
	if err := w.Commit(); err != nil {
		return errors.WithMessagef(err, "committing tx for txid [%s] failed", record.Anchor)
	}

	logger.DebugfContext(ctx, "appending new records completed without errors")

	return nil
}

// Transactions returns an iterators of transaction records filtered by the given params.
func (d *StoreService) Transactions(ctx context.Context, params QueryTransactionsParams, pagination Pagination) (*PageTransactionsIterator, error) {
	return d.db.QueryTransactions(ctx, params, pagination)
}

// TokenRequests returns an iterator over the token requests matching the passed params
func (d *StoreService) TokenRequests(ctx context.Context, params QueryTokenRequestsParams) (dbdriver.TokenRequestIterator, error) {
	return d.db.QueryTokenRequests(ctx, params)
}

// NewPaymentsFilter returns a programmable filter over the payments sent or received by enrollment IDs.
func (d *StoreService) NewPaymentsFilter() *PaymentsFilter {
	return &PaymentsFilter{
		db: d,
	}
}

// NewHoldingsFilter returns a programmable filter over the holdings owned by enrollment IDs.
func (d *StoreService) NewHoldingsFilter() *HoldingsFilter {
	return &HoldingsFilter{
		db: d,
	}
}

// SetStatus sets the status of the audit records with the passed transaction id to the passed status
func (d *StoreService) SetStatus(ctx context.Context, txID string, status dbdriver.TxStatus, message string) error {
	logger.DebugfContext(ctx, "set status [%s][%s]...", txID, status)
	if err := d.db.SetStatus(ctx, txID, status, message); err != nil {
		return errors.Wrapf(err, "failed setting status [%s][%s]", txID, dbdriver.TxStatusMessage[status])
	}

	// notify the listeners
	d.Notify(common.StatusEvent{
		Ctx:            ctx,
		TxID:           txID,
		ValidationCode: status,
	})
	logger.DebugfContext(ctx, "set status [%s][%s]...done without errors", txID, dbdriver.TxStatusMessage[status])

	return nil
}

// GetStatus return the status of the given transaction id.
// It returns an error if no transaction with that id is found
func (d *StoreService) GetStatus(ctx context.Context, txID string) (TxStatus, string, error) {
	logger.DebugfContext(ctx, "get status [%s]...", txID)
	status, message, err := d.db.GetStatus(ctx, txID)
	if err != nil {
		return Unknown, "", errors.Wrapf(err, "failed getting status [%s]", txID)
	}
	logger.DebugfContext(ctx, "Got status [%s][%s]", txID, status)

	return status, message, nil
}

// GetTokenRequest returns the token request bound to the passed transaction id, if available.
func (d *StoreService) GetTokenRequest(ctx context.Context, txID string) ([]byte, error) {
	return d.db.GetTokenRequest(ctx, txID)
}

// GetTokenRequests returns the token requests bound to the given tx ids in
// a single query. See driver.TransactionStore.GetTokenRequests for details
// about missing-key semantics.
func (d *StoreService) GetTokenRequests(ctx context.Context, txIDs []string) (map[string][]byte, error) {
	return d.db.GetTokenRequests(ctx, txIDs)
}

// AcquireLocks acquires locks for the passed anchor and enrollment ids.
// This can be used to prevent concurrent read/write access to the audit records of the passed enrollment ids.
func (d *StoreService) AcquireLocks(ctx context.Context, anchor string, eIDs ...string) error {
	// This implementation allows concurrent calls to AcquireLocks such that if two
	// or more calls involve non-overlapping enrollment IDs, both calls will succeed.
	// To achieve this, we first remove any duplicates from the list of enrollment IDs.
	// Next, we sort this list. Sorting ensures that two concurrent invocations of
	// AcquireLocks with intersecting sets of enrollment IDs do not result in a deadlock.
	// For example, consider a scenario where one invocation attempts to lock (Alice, Bob)
	// and another tries to lock (Bob, Alice).
	// Without sorting, these two calls could deadlock. Sorting prevents this issue.
	dedup := deduplicateAndSort(eIDs)
	logger.DebugfContext(ctx, "Acquire locks for [%s:%v] enrollment ids", anchor, dedup)
	d.eIDsLocks.LoadOrStore(anchor, dedup)
	for _, id := range dedup {
		lock, _ := d.eIDsLocks.LoadOrStore(id, &sync.RWMutex{})
		lock.(*sync.RWMutex).Lock()
		logger.DebugfContext(ctx, "Acquire locks for [%s:%v] enrollment id done", anchor, id)
	}
	logger.DebugfContext(ctx, "Acquire locks for [%s:%v] enrollment ids...done", anchor, dedup)

	return nil
}

// ReleaseLocks releases the locks associated to the passed anchor
func (d *StoreService) ReleaseLocks(ctx context.Context, anchor string) {
	dedupBoxed, ok := d.eIDsLocks.LoadAndDelete(anchor)
	if !ok {
		logger.DebugfContext(ctx, "nothing to release for [%s] ", anchor)

		return
	}
	dedup := dedupBoxed.([]string)
	logger.DebugfContext(ctx, "Release locks for [%s:%v] enrollment ids", anchor, dedup)
	for _, id := range dedup {
		lock, ok := d.eIDsLocks.Load(id)
		if !ok {
			logger.Warnf("unlock for enrollment id [%s:%s] not possible, lock never acquired", anchor, id)

			continue
		}
		logger.DebugfContext(ctx, "unlock lock for [%s:%v] enrollment id done", anchor, id)
		lock.(*sync.RWMutex).Unlock()
	}
	logger.DebugfContext(ctx, "Release locks for [%s:%v] enrollment ids...done", anchor, dedup)
}

// AcquireRecoveryLeadership tries to acquire the DB-backed recovery leadership lease.
func (d *StoreService) AcquireRecoveryLeadership(ctx context.Context, lockID int64) (dbdriver.RecoveryLeadership, bool, error) {
	leadership, acquired, err := d.db.AcquireRecoveryLeadership(ctx, lockID)
	if err != nil {
		return nil, false, errors.Wrapf(err, "failed to acquire recovery leadership")
	}

	return leadership, acquired, nil
}

// ClaimPendingTransactions returns a claimed batch of Pending transactions older than the given duration.
func (d *StoreService) ClaimPendingTransactions(ctx context.Context, olderThan time.Duration, leaseDuration time.Duration, limit int, owner string) ([]*TransactionRecord, error) {
	storedBefore := time.Now().UTC().Add(-olderThan)
	logger.DebugfContext(ctx, "claiming pending transactions stored before %s (older than %s), lease duration [%s], limit [%d], owner [%s]",
		storedBefore, olderThan, leaseDuration, limit, owner)

	records, err := d.db.ClaimPendingTransactions(ctx, dbdriver.RecoveryClaimParams{
		OlderThan:     storedBefore,
		LeaseDuration: leaseDuration,
		Limit:         limit,
		Owner:         owner,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to claim pending transactions")
	}

	logger.DebugfContext(ctx, "claimed %d pending transactions older than %s for owner [%s]", len(records), olderThan, owner)

	return records, nil
}

// ReleaseRecoveryClaim clears the recovery claim for the passed transaction if owned by owner.
// The message parameter is stored for audit/debugging purposes.
func (d *StoreService) ReleaseRecoveryClaim(ctx context.Context, txID string, owner string, message string) error {
	if err := d.db.ReleaseRecoveryClaim(ctx, txID, owner, message); err != nil {
		return errors.Wrapf(err, "failed to release recovery claim for tx [%s]", txID)
	}

	return nil
}

// deduplicateAndSort removes duplicate entries from a slice and sort it
func deduplicateAndSort(source []string) []string {
	slice := collections.NewSet(source...).ToSlice()
	slices.Sort(slice)

	return slice
}
