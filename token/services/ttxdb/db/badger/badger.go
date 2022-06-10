/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package badger

import (
	"bytes"
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb/db/badger/keys"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb/driver"

	"github.com/dgraph-io/badger/v3"
	"github.com/dgraph-io/ristretto/z"
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

type MovementRecord struct {
	Id     uint64
	Record *driver.MovementRecord
}

type TransactionRecord struct {
	Id     uint64
	Record *driver.TransactionRecord
}

type Persistence struct {
	db          *badger.DB
	numGoStream int
	seq         *badger.Sequence
	txn         *badger.Txn
	txnLock     sync.Mutex
}

func OpenDB(path string) (*Persistence, error) {
	db, err := badger.Open(badger.DefaultOptions(path))
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

	// TODO: what to do with db.txn if it's not nil?

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

	bytes, err := MarshalMovementRecord(value)
	if err != nil {
		return errors.Wrapf(err, "could not marshal record for key %s", key)
	}

	err = db.txn.Set([]byte(key), bytes)
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

	bytes, err := MarshalTransactionRecord(value)
	if err != nil {
		return errors.Wrapf(err, "could not marshal record for key %s", key)
	}

	err = db.txn.Set([]byte(key), bytes)
	if err != nil {
		return errors.Wrapf(err, "could not set value for key %s", key)
	}

	return nil
}

func (db *Persistence) QueryTransactions(from, to *time.Time) (driver.TransactionIterator, error) {
	txn := db.db.NewTransaction(false)
	it := txn.NewIterator(badger.DefaultIteratorOptions)
	it.Seek([]byte("tx"))

	return &TransactionIterator{it: it, from: from, to: to}, nil
}

func (db *Persistence) SetStatus(txID string, status driver.TxStatus) error {
	// search for all matching keys
	type Entry struct {
		key   string
		value []byte
	}
	var entries []Entry
	stream := db.db.NewStream()
	stream.NumGo = db.numGoStream
	stream.LogPrefix = streamLogPrefixStatus
	txIdAsBytes := []byte(txID)
	stream.ChooseKey = func(item *badger.Item) bool {
		return bytes.HasSuffix(item.Key(), txIdAsBytes)
	}
	stream.Send = func(buf *z.Buffer) error {
		list, err := badger.BufferToKVList(buf)
		if err != nil {
			return err
		}
		for _, kv := range list.Kv {
			entries = append(entries, Entry{key: string(kv.Key), value: kv.Value})
		}
		return nil

	}
	if err := stream.Orchestrate(context.Background()); err != nil {
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
		var bytes []byte
		switch {
		case strings.HasPrefix(entry.key, "mv"):
			record, err := UnmarshalMovementRecord(entry.value)
			if err != nil {
				return errors.Wrapf(err, "could not unmarshal key %s", entry.key)
			}
			record.Record.Status = status
			bytes, err = MarshalMovementRecord(record)
			if err != nil {
				return errors.Wrapf(err, "could not marshal record for key %s", entry.key)
			}
		case strings.HasPrefix(entry.key, "tx"):
			record, err := UnmarshalTransactionRecord(entry.value)
			if err != nil {
				return errors.Wrapf(err, "could not unmarshal key %s", entry.key)
			}
			record.Record.Status = status
			bytes, err = MarshalTransactionRecord(record)
			if err != nil {
				return errors.Wrapf(err, "could not marshal record for key %s", entry.key)
			}
		default:
			continue
		}

		logger.Debugf("setting key %s to %s", entry.key, string(bytes))
		if err := txn.Set([]byte(entry.key), bytes); err != nil {
			return errors.Wrapf(err, "could not set value for key %s", entry.key)
		}
	}
	if err := txn.Commit(); err != nil {
		txn.Discard()
		return errors.Wrapf(err, "could not commit transaction to set status for tx %s", txID)
	}
	return nil
}

func (db *Persistence) QueryMovements(enrollmentIDs []string, tokenTypes []string, txStatuses []driver.TxStatus, searchDirection driver.SearchDirection, movementDirection driver.MovementDirection, numRecords int) ([]*driver.MovementRecord, error) {
	// TODO: Move to stream
	txn := db.db.NewTransaction(false)
	it := txn.NewIterator(badger.DefaultIteratorOptions)
	var records RecordSlice
	defer it.Close()

	selector := &MovementSelector{
		enrollmentIDs:     enrollmentIDs,
		tokenTypes:        tokenTypes,
		txStatuses:        txStatuses,
		searchDirection:   searchDirection,
		movementDirection: movementDirection,
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
	switch searchDirection {
	case driver.FromBeginning:
		sort.Sort(records)
	case driver.FromLast:
		sort.Sort(sort.Reverse(records))
	}

	if numRecords > 0 && len(records) > numRecords {
		records = records[:numRecords]
	}

	var res []*driver.MovementRecord
	for _, record := range records {
		res = append(res, record.Record)
	}

	return res, nil
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

func dbKey(namespace, key string) string {
	return namespace + keys.NamespaceSeparator + key
}

type RecordSlice []*MovementRecord

func (p RecordSlice) Len() int           { return len(p) }
func (p RecordSlice) Less(i, j int) bool { return p[i].Id < p[j].Id }
func (p RecordSlice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

type TransactionIterator struct {
	db   *Persistence
	it   *badger.Iterator
	from *time.Time
	to   *time.Time
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

		// is record in the time range
		if t.from != nil && record.Record.Timestamp.Before(*t.from) {
			continue
		}
		if t.to != nil && record.Record.Timestamp.After(*t.to) {
			return nil, nil
		}
		logger.Debugf("found transaction [%s,%s]", string(item.Key()), record.Record.TxID)
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
	enrollmentIDs     []string
	tokenTypes        []string
	txStatuses        []driver.TxStatus
	searchDirection   driver.SearchDirection
	movementDirection driver.MovementDirection
}

// Select returns true is the record matches the selection criteria
func (m *MovementSelector) Select(record *MovementRecord) bool {
	if len(m.enrollmentIDs) != 0 {
		found := false
		for _, id := range m.enrollmentIDs {
			if record.Record.EnrollmentID == id {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if len(m.tokenTypes) != 0 {
		found := false
		for _, typ := range m.tokenTypes {
			if record.Record.TokenType == typ {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if len(m.txStatuses) != 0 {
		found := false
		for _, st := range m.txStatuses {
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

	if m.movementDirection == driver.Sent && record.Record.Amount.Sign() > 0 {
		return false
	}

	if m.movementDirection == driver.Received && record.Record.Amount.Sign() < 0 {
		return false
	}

	return true
}
