/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package badger

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	"github.com/dgraph-io/badger/v3"
	"github.com/dgraph-io/ristretto/z"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb/db/badger/keys"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb/driver"
	"github.com/pkg/errors"
)

const (
	// SeqBandwidth sets the size of the lease, determining how many Next() requests can be served from memory
	SeqBandwidth = 10
	// IndexLength is the length of the index used to store the sequence
	IndexLength = 26
	// DefaultNumGoStream is the default number of goroutines used to process the DB streams
	DefaultNumGoStream = 16
	// streamLogPrefixStatus is the prefix for the status log
	streamLogPrefixStatus = "ttxdb.SetStatus"
)

type TransactionRecordSelector interface {
	Select(record *TransactionRecord) (bool, bool)
}

type ValidationRecordSelector interface {
	Select(record *ValidationRecord) (bool, bool)
}

type MovementRecord struct {
	Id     uint64
	Record *driver.MovementRecord
}

type TransactionRecord struct {
	Id     uint64
	Record *driver.TransactionRecord
}

type ValidationRecord struct {
	Id     uint64
	Record *driver.ValidationRecord
}

type TokenRequest struct {
	TxID string
	TR   []byte
}

type TransactionEndorseAck struct {
	TxID  string
	ID    view.Identity
	Sigma []byte
}

type Persistence struct {
	db          *badger.DB
	numGoStream int
	seq         *badger.Sequence
	txn         *badger.Txn
	txnLock     sync.Mutex
}

func OpenDB(path string) (*Persistence, error) {
	info, err := os.Stat(path)
	logger.Debugf("Opening TTX DB at [%s][%s:%s]", path, info, err)

	opts := badger.DefaultOptions(path)
	opts.Logger = logger
	db, err := badger.Open(opts)
	if err != nil {
		return nil, errors.Wrapf(err, "could not open DB at '%s'", path)
	}
	seq, err := db.GetSequence([]byte("idseq"), SeqBandwidth)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting sequence for DB at '%s'", path)
	}

	return &Persistence{db: db, seq: seq, numGoStream: DefaultNumGoStream}, nil
}

func (db *Persistence) Close() error {
	db.txnLock.Lock()
	defer db.txnLock.Unlock()

	// discard current transaction, if any
	if db.txn != nil {
		db.txn.Discard()
		db.txn = nil
	}

	if err := db.seq.Release(); err != nil {
		logger.Errorf("failed closing seq [%s]", err)
	}

	err := db.db.Close()
	if err != nil {
		return errors.Wrap(err, "could not close DB")
	}

	return nil
}

func (db *Persistence) BeginUpdate() error {
	db.txnLock.Lock()
	defer db.txnLock.Unlock()

	if db.txn != nil {
		return errors.New("previous commit in progress")
	}

	db.txn = db.db.NewTransaction(true)

	return nil
}

func (db *Persistence) Commit() error {
	db.txnLock.Lock()
	defer db.txnLock.Unlock()

	if db.txn == nil {
		return errors.New("no commit in progress")
	}

	err := db.txn.Commit()
	if err != nil {
		return errors.Wrap(err, "could not commit transaction")
	}

	db.txn = nil

	return nil
}

func (db *Persistence) Discard() error {
	db.txnLock.Lock()
	defer db.txnLock.Unlock()

	if db.txn == nil {
		return errors.New("no commit in progress")
	}

	db.txn.Discard()

	db.txn = nil

	return nil
}

func (db *Persistence) AddMovement(record *driver.MovementRecord) error {
	logger.Debugf("Adding movement record [%s:%s:%s:%s]", record.TxID, record.TokenType, record.EnrollmentID, record.Amount)
	next, key, err := db.movementKey(record.TxID)
	if err != nil {
		return errors.Wrapf(err, "could not get key for movement %s", record.TxID)
	}

	value := &MovementRecord{
		Id:     next,
		Record: record,
	}

	b, err := MarshalMovementRecord(value)
	if err != nil {
		return errors.Wrapf(err, "could not marshal record for key %s", key)
	}

	err = db.txn.Set([]byte(key), b)
	if err != nil {
		return errors.Wrapf(err, "could not set value for key %s", key)
	}

	return nil
}

func (db *Persistence) AddTransaction(record *driver.TransactionRecord) error {
	next, key, err := db.transactionKey(record.TxID)
	if err != nil {
		return errors.Wrapf(err, "could not get key for transaction %s", record.TxID)
	}

	value := &TransactionRecord{
		Id:     next,
		Record: record,
	}
	logger.Debugf("Adding transaction record [%s:%d:%s:%s:%s:%s]", record.TxID, record.ActionType, record.TokenType, record.SenderEID, record.RecipientEID, record.Amount)

	b, err := MarshalTransactionRecord(value)
	if err != nil {
		return errors.Wrapf(err, "could not marshal record for key %s", key)
	}

	err = db.txn.Set([]byte(key), b)
	if err != nil {
		return errors.Wrapf(err, "could not set value for key %s", key)
	}

	return nil
}

func (db *Persistence) AddValidationRecord(txID string, tr []byte, meta map[string][]byte) error {
	next, key, err := db.validationRecordKey(txID)
	if err != nil {
		return errors.Wrapf(err, "could not get key for validation record %s", txID)
	}
	logger.Debugf("Adding validation record [%s] with key [%s]", txID, key)

	value := &ValidationRecord{
		Id: next,
		Record: &driver.ValidationRecord{
			TxID:         txID,
			Timestamp:    time.Now(),
			TokenRequest: tr,
			Metadata:     meta,
		},
	}

	b, err := MarshalValidationRecord(value)
	if err != nil {
		return errors.Wrapf(err, "could not marshal record for key %s", key)
	}

	err = db.txn.Set([]byte(key), b)
	if err != nil {
		return errors.Wrapf(err, "could not set value for key %s", key)
	}

	return nil
}

func (db *Persistence) AddTokenRequest(txID string, tr []byte) error {
	key, err := db.tokenRequestKey(txID)
	if err != nil {
		return errors.Wrapf(err, "could not get key for token request %s", txID)
	}
	logger.Debugf("Adding token request [%s] with key [%s]", txID, key)

	value := &TokenRequest{
		TxID: txID,
		TR:   tr,
	}

	b, err := MarshalTokenRequest(value)
	if err != nil {
		return errors.Wrapf(err, "could not marshal record for key %s", key)
	}

	err = db.txn.Set([]byte(key), b)
	if err != nil {
		return errors.Wrapf(err, "could not set value for key %s", key)
	}

	return nil
}

func (db *Persistence) GetTokenRequest(txID string) ([]byte, error) {
	key, err := db.tokenRequestKey(txID)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get key for token request %s", txID)
	}

	txn := db.db.NewTransaction(false)
	defer txn.Discard()
	item, err := txn.Get([]byte(key))
	if err != nil {
		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "could not set value for key %s", key)
	}
	value := &TokenRequest{}
	if err := item.Value(func(val []byte) error {
		var err error
		value, err = UnmarshalTokenRequest(val)
		if err != nil {
			return errors.Wrapf(err, "could not set value for key %s", key)
		}
		return nil
	}); err != nil {
		return nil, errors.Wrapf(err, "failed to load token request for %s", txID)
	}
	return value.TR, nil
}

func (db *Persistence) QueryTransactions(params driver.QueryTransactionsParams) (driver.TransactionIterator, error) {
	txn := db.db.NewTransaction(false)
	it := txn.NewIterator(badger.DefaultIteratorOptions)
	it.Seek([]byte("tx"))

	selector := &TransactionSelector{
		params: params,
	}
	return &TransactionIterator{it: it, selector: selector}, nil
}

func (db *Persistence) QueryMovements(params driver.QueryMovementsParams) ([]*driver.MovementRecord, error) {
	// TODO: Move to stream
	txn := db.db.NewTransaction(false)
	it := txn.NewIterator(badger.DefaultIteratorOptions)
	var records RecordSlice
	defer it.Close()

	selector := &MovementSelector{
		params: params,
	}
	for it.Rewind(); it.Valid(); it.Next() {
		item := it.Item()
		if !strings.HasPrefix(string(item.Key()), "mv") {
			continue
		}
		var record *MovementRecord
		err := item.Value(func(val []byte) error {
			if len(val) == 0 {
				record = nil
				return nil
			}
			var err error
			if record, err = UnmarshalMovementRecord(val); err != nil {
				return errors.Wrapf(err, "could not unmarshal key %s", string(item.Key()))
			}
			return nil
		})
		if record == nil {
			continue
		}
		if err != nil {
			return nil, errors.Wrapf(err, "could not get movementDirection for key %s", string(item.Key()))
		}

		// filter
		if selector.Select(record) {
			records = append(records, record)
		}
	}

	// Sort
	switch params.SearchDirection {
	case driver.FromBeginning:
		sort.Sort(records)
	case driver.FromLast:
		sort.Sort(sort.Reverse(records))
	}

	if params.NumRecords > 0 && len(records) > params.NumRecords {
		records = records[:params.NumRecords]
	}

	var res []*driver.MovementRecord
	for _, record := range records {
		res = append(res, record.Record)
	}

	return res, nil
}

func (db *Persistence) QueryValidations(params driver.QueryValidationRecordsParams) (driver.ValidationRecordsIterator, error) {
	txn := db.db.NewTransaction(false)
	it := txn.NewIterator(badger.DefaultIteratorOptions)
	it.Seek([]byte("mt"))

	selector := &ValidationRecordsSelector{
		params: params,
	}
	return &ValidationRecordsIterator{it: it, selector: selector}, nil
}

type persistenceEntry struct {
	key   string
	value []byte
}

func (db *Persistence) entriesByTxID(condition func([]byte, []byte) bool, txID string, prefix string) ([]persistenceEntry, error) {
	var entries []persistenceEntry
	stream := db.db.NewStream()
	stream.NumGo = db.numGoStream
	stream.LogPrefix = streamLogPrefixStatus
	if len(prefix) > 0 {
		stream.Prefix = []byte(prefix)
	}
	txIdAsBytes := []byte(txID)
	stream.ChooseKey = func(item *badger.Item) bool {
		fmt.Printf("key [%s]\n", item.Key())
		return condition(item.Key(), txIdAsBytes)
	}
	stream.Send = func(buf *z.Buffer) error {
		list, err := badger.BufferToKVList(buf)
		if err != nil {
			return err
		}
		for _, kv := range list.Kv {
			entries = append(entries, persistenceEntry{key: string(kv.Key), value: kv.Value})
		}
		return nil

	}
	if err := stream.Orchestrate(context.Background()); err != nil {
		return nil, err
	}
	return entries, nil
}

func (db *Persistence) SetStatus(txID string, status driver.TxStatus) error {
	logger.Debugf("set status of [%s] to [%s]", txID, status)
	entries, err := db.entriesByTxID(bytes.HasSuffix, txID, "")
	if err != nil {
		return err
	}

	if len(entries) == 0 {
		// nothing to update
		logger.Debugf("no entries found for txID %s, skipping", txID)
		return nil
	}

	// update status for all matching keys
	txn := db.db.NewTransaction(true)
	for _, entry := range entries {
		var b []byte
		switch {
		case strings.HasPrefix(entry.key, "mv"):
			logger.Debugf("set status of movement [%s] to [%s]", txID, status)
			record, err := UnmarshalMovementRecord(entry.value)
			if err != nil {
				return errors.Wrapf(err, "could not unmarshal key %s", entry.key)
			}
			record.Record.Status = status
			b, err = MarshalMovementRecord(record)
			if err != nil {
				return errors.Wrapf(err, "could not marshal record for key %s", entry.key)
			}
		case strings.HasPrefix(entry.key, "tx"):
			logger.Debugf("set status of transaction [%s] to [%s]", txID, status)
			record, err := UnmarshalTransactionRecord(entry.value)
			if err != nil {
				return errors.Wrapf(err, "could not unmarshal key %s", entry.key)
			}
			record.Record.Status = status
			b, err = MarshalTransactionRecord(record)
			if err != nil {
				return errors.Wrapf(err, "could not marshal record for key %s", entry.key)
			}
		case strings.HasPrefix(entry.key, "mt"):
			logger.Debugf("set status of validation record [%s] to [%s]", txID, status)
			record, err := UnmarshalValidationRecord(entry.value)
			if err != nil {
				return errors.Wrapf(err, "could not unmarshal key %s", entry.key)
			}
			record.Record.Status = status
			b, err = MarshalValidationRecord(record)
			if err != nil {
				return errors.Wrapf(err, "could not marshal record for key %s", entry.key)
			}
		default:
			continue
		}

		logger.Debugf("setting key %s to %s", entry.key, string(b))
		if err := txn.Set([]byte(entry.key), b); err != nil {
			return errors.Wrapf(err, "could not set value for key %s", entry.key)
		}
	}
	if err := txn.Commit(); err != nil {
		txn.Discard()
		return errors.Wrapf(err, "could not commit transaction to set status for tx %s", txID)
	}
	return nil
}

func (db *Persistence) GetStatus(txID string) (driver.TxStatus, error) {
	entries, err := db.entriesByTxID(bytes.HasSuffix, txID, "tx")
	if err != nil {
		return driver.Unknown, err
	}

	if len(entries) == 0 {
		// nothing to update
		logger.Debugf("no entries found for txID %s, skipping", txID)
		return driver.Unknown, nil
	}

	entry := entries[0]
	record, err := UnmarshalTransactionRecord(entry.value)
	if err != nil {
		return driver.Unknown, errors.Wrapf(err, "could not unmarshal key %s", entry.key)
	}
	return record.Record.Status, nil
}

func (db *Persistence) AddTransactionEndorsementAck(txID string, id view.Identity, sigma []byte) error {
	key, err := db.transactionEndorseAckKey(txID, id)
	if err != nil {
		return errors.Wrapf(err, "could not get key for token request %s", txID)
	}
	logger.Debugf("adding transaction endorsement ack [%s] with key [%s]", txID, key)

	value := &TransactionEndorseAck{
		TxID:  txID,
		ID:    id,
		Sigma: sigma,
	}

	b, err := MarshalTransactionEndorseAck(value)
	if err != nil {
		return errors.Wrapf(err, "could not marshal record for key %s", key)
	}

	txn := db.db.NewTransaction(true)
	defer txn.Discard()
	err = txn.Set([]byte(key), b)
	if err != nil {
		return errors.Wrapf(err, "could not set value for key %s", key)
	}
	err = txn.Commit()
	if err != nil {
		return errors.Wrapf(err, "could not commit transaction for [%s]", txID)
	}

	return nil
}

func (db *Persistence) GetTransactionEndorsementAcks(txID string) (map[string][]byte, error) {
	entries, err := db.entriesByTxID(bytes.Contains, txID, "tea")
	if err != nil {
		return nil, err
	}

	if len(entries) == 0 {
		// nothing to update
		logger.Debugf("no entries found for txID %s, skipping", txID)
		return nil, nil
	}

	acks := make(map[string][]byte, len(entries))
	for _, entry := range entries {
		record, err := UnmarshalTransactionEndorseAck(entry.value)
		if err != nil {
			return nil, errors.Wrapf(err, "could not unmarshal key %s", entry.key)
		}
		acks[record.ID.String()] = record.Sigma
	}
	return acks, nil
}

func (db *Persistence) transactionKey(txID string) (uint64, string, error) {
	next, err := db.seq.Next()
	if err != nil {
		return 0, "", errors.Wrapf(err, "failed getting next index")
	}
	return next, dbKey("tx", dbKey(kThLexicographicString(IndexLength, int(next)), txID)), nil
}

func (db *Persistence) movementKey(txID string) (uint64, string, error) {
	next, err := db.seq.Next()
	if err != nil {
		return 0, "", errors.Wrapf(err, "failed getting next index")
	}
	return next, dbKey("mv", dbKey(kThLexicographicString(IndexLength, int(next)), txID)), nil
}

func (db *Persistence) validationRecordKey(txID string) (uint64, string, error) {
	next, err := db.seq.Next()
	if err != nil {
		return 0, "", errors.Wrapf(err, "failed getting next index")
	}
	return next, dbKey("mt", dbKey(kThLexicographicString(IndexLength, int(next)), txID)), nil
}

func (db *Persistence) tokenRequestKey(txID string) (string, error) {
	return dbKey("tr", txID), nil
}

func (db *Persistence) transactionEndorseAckKey(txID string, id view.Identity) (string, error) {
	return dbKey("tea", dbKey(txID, id.String())), nil
}

func dbKey(namespace, key string) string {
	return namespace + keys.NamespaceSeparator + key
}

type RecordSlice []*MovementRecord

func (p RecordSlice) Len() int           { return len(p) }
func (p RecordSlice) Less(i, j int) bool { return p[i].Id < p[j].Id }
func (p RecordSlice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

type TransactionIterator struct {
	it       *badger.Iterator
	selector TransactionRecordSelector
}

func (t *TransactionIterator) Close() {
	t.it.Close()
}

func (t *TransactionIterator) Next() (*driver.TransactionRecord, error) {
	for {
		if !t.it.Valid() {
			return nil, nil
		}
		item := t.it.Item()
		if item == nil {
			return nil, nil
		}

		if !strings.HasPrefix(string(item.Key()), "tx") {
			t.it.Next()
			continue
		}

		var record *TransactionRecord
		err := item.Value(func(val []byte) error {
			var err error
			if record, err = UnmarshalTransactionRecord(val); err != nil {
				return errors.Wrapf(err, "could not unmarshal key %s", string(item.Key()))
			}
			return nil
		})
		if err != nil {
			return nil, errors.Wrapf(err, "could not get transaction for key %s", string(item.Key()))
		}

		t.it.Next()

		matched, stop := t.selector.Select(record)
		if stop {
			return nil, nil
		}
		if !matched {
			continue
		}
		logger.Debugf("found transaction [%s,%s][%s,%s]", string(item.Key()), record.Record.TxID, record.Record.SenderEID, record.Record.RecipientEID)
		return record.Record, nil
	}
}

// kThLexicographicString returns the k-th string of length n over alphabet (a+25) in lexicographic order.
func kThLexicographicString(n, k int) string {
	//k += 4
	d := make([]int, n)
	for i := n - 1; i > -1; i-- {
		d[i] = k % 26
		k /= 26
	}
	if k > 0 {
		return "-1"
	}
	var sb strings.Builder
	for i := 0; i < n; i++ {
		sb.WriteRune(rune(d[i] + ('a')))
	}
	return sb.String()
}

// MovementSelector is used to select a set of movement records
type MovementSelector struct {
	params driver.QueryMovementsParams
}

// Select returns true is the record matches the selection criteria
func (m *MovementSelector) Select(record *MovementRecord) bool {
	if len(m.params.EnrollmentIDs) != 0 {
		found := false
		for _, id := range m.params.EnrollmentIDs {
			if record.Record.EnrollmentID == id {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if len(m.params.TokenTypes) != 0 {
		found := false
		for _, typ := range m.params.TokenTypes {
			if record.Record.TokenType == typ {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if len(m.params.TxStatuses) != 0 {
		found := false
		for _, st := range m.params.TxStatuses {
			if record.Record.Status == st {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	} else {
		// exclude the deleted
		if record.Record.Status == driver.Deleted {
			return false
		}
	}

	if m.params.MovementDirection == driver.Sent && record.Record.Amount.Sign() > 0 {
		return false
	}

	if m.params.MovementDirection == driver.Received && record.Record.Amount.Sign() < 0 {
		return false
	}

	return true
}

// TransactionSelector is used to select a set of transaction records
type TransactionSelector struct {
	params driver.QueryTransactionsParams
}

// Select returns true is the record matches the selection criteria.
// Additionally, it returns another flag indicating if it is time to stop or not.
func (t *TransactionSelector) Select(record *TransactionRecord) (bool, bool) {
	// match the time constraints
	if t.params.From != nil && record.Record.Timestamp.Before(*t.params.From) {
		logger.Debugf("skipping transaction [%s] because it is before the from time", record.Record.TxID)
		return false, false
	}
	if t.params.To != nil && record.Record.Timestamp.After(*t.params.To) {
		logger.Debugf("skipping transaction [%s] because it is after the to time", record.Record.TxID)
		return false, true
	}
	if len(t.params.ActionTypes) != 0 {
		found := false
		for _, actionType := range t.params.ActionTypes {
			if actionType == record.Record.ActionType {
				found = true
				break
			}
		}
		if !found {
			return false, false
		}
	}
	if len(t.params.Statuses) != 0 {
		found := false
		for _, statusType := range t.params.Statuses {
			if statusType == record.Record.Status {
				found = true
				break
			}
		}
		if !found {
			return false, false
		}
	}
	// match the wallet
	senderMatch := true
	if len(t.params.SenderWallet) != 0 && record.Record.SenderEID != t.params.SenderWallet {
		senderMatch = false
	}
	receiverMatch := true
	if len(t.params.RecipientWallet) != 0 && record.Record.RecipientEID != t.params.RecipientWallet {
		receiverMatch = false
	}
	if !senderMatch && !receiverMatch {
		logger.Debugf("skipping transaction [%s] because it does not match the sender [%s:%s] or receiver [%s:%s]",
			record.Record.TxID,
			record.Record.SenderEID, t.params.SenderWallet,
			record.Record.RecipientEID, t.params.RecipientWallet,
		)
		return false, false
	}
	return true, false
}

// ValidationRecordsSelector is used to select a set of transaction records
type ValidationRecordsSelector struct {
	params driver.QueryValidationRecordsParams
}

// Select returns true is the record matches the selection criteria.
// Additionally, it returns another flag indicating if it is time to stop or not.
func (t *ValidationRecordsSelector) Select(record *ValidationRecord) (bool, bool) {
	// match the time constraints
	if t.params.From != nil && record.Record.Timestamp.Before(*t.params.From) {
		logger.Debugf("skipping transaction [%s] because it is before the from time", record.Record.TxID)
		return false, false
	}
	if t.params.To != nil && record.Record.Timestamp.After(*t.params.To) {
		logger.Debugf("skipping transaction [%s] because it is after the to time", record.Record.TxID)
		return false, true
	}

	if len(t.params.Statuses) != 0 {
		found := false
		for _, statusType := range t.params.Statuses {
			if statusType == record.Record.Status {
				found = true
				break
			}
		}
		if !found {
			return false, false
		}
	}

	if t.params.Filter != nil {
		if !t.params.Filter(record.Record) {
			return false, false
		}
	}

	return true, false
}

type ValidationRecordsIterator struct {
	it       *badger.Iterator
	selector ValidationRecordSelector
}

func (t *ValidationRecordsIterator) Close() {
	t.it.Close()
}

func (t *ValidationRecordsIterator) Next() (*driver.ValidationRecord, error) {
	for {
		if !t.it.Valid() {
			return nil, nil
		}
		item := t.it.Item()
		if item == nil {
			return nil, nil
		}

		if !strings.HasPrefix(string(item.Key()), "mt") {
			t.it.Next()
			continue
		}

		var record *ValidationRecord
		err := item.Value(func(val []byte) error {
			var err error
			if record, err = UnmarshalValidationRecord(val); err != nil {
				return errors.Wrapf(err, "could not unmarshal key %s", string(item.Key()))
			}
			return nil
		})
		if err != nil {
			return nil, errors.Wrapf(err, "could not get ValidationRecords for key %s", string(item.Key()))
		}

		t.it.Next()

		matched, stop := t.selector.Select(record)
		if stop {
			return nil, nil
		}
		if !matched {
			continue
		}
		logger.Debugf("found validation record [%s,%s]", string(item.Key()), record.Record.TxID)
		return record.Record, nil
	}
}
