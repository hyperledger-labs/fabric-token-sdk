/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package badger

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v3"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditor/auditdb/db/badger/keys"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditor/auditdb/driver"
	"github.com/pkg/errors"
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
	db *badger.DB

	seq     *badger.Sequence
	txn     *badger.Txn
	txnLock sync.Mutex
}

func OpenDB(path string) (*Persistence, error) {
	db, err := badger.Open(badger.DefaultOptions(path))
	if err != nil {
		return nil, errors.Wrapf(err, "could not open DB at '%s'", path)
	}
	seq, err := db.GetSequence([]byte("idseq"), 1)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting sequence for DB at '%s'", path)
	}

	return &Persistence{db: db, seq: seq}, nil
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
	next, err := db.seq.Next()
	if err != nil {
		return errors.Wrapf(err, "failed getting next index")
	}
	dbKey := dbKey("mv", dbKey(fmt.Sprintf("%d", next), record.TxID))

	value := &MovementRecord{
		Id:     next,
		Record: record,
	}

	bytes, err := json.Marshal(value)
	if err != nil {
		return errors.Wrapf(err, "could not marshal record for key %s", dbKey)
	}

	err = db.txn.Set([]byte(dbKey), bytes)
	if err != nil {
		return errors.Wrapf(err, "could not set value for key %s", dbKey)
	}

	return nil
}

func (db *Persistence) AddTransaction(record *driver.TransactionRecord) error {
	next, err := db.seq.Next()
	if err != nil {
		return errors.Wrapf(err, "failed getting next index")
	}
	dbKey := dbKey("tx", dbKey(fmt.Sprintf("%d", next), record.TxID))

	value := &TransactionRecord{
		Id:     next,
		Record: record,
	}

	bytes, err := json.Marshal(value)
	if err != nil {
		return errors.Wrapf(err, "could not marshal record for key %s", dbKey)
	}

	err = db.txn.Set([]byte(dbKey), bytes)
	if err != nil {
		return errors.Wrapf(err, "could not set value for key %s", dbKey)
	}

	return nil
}

func (db *Persistence) QueryTransactions(from, to *time.Time) (driver.TransactionIterator, error) {
	txn := db.db.NewTransaction(false)
	it := txn.NewIterator(badger.DefaultIteratorOptions)
	it.Rewind()
	return &TransactionIterator{it: it, from: from, to: to}, nil
}

func (db *Persistence) SetStatus(txID string, status driver.TxStatus) error {
	// Search the records for the passed transaction ID and update the status
	it := db.txn.NewIterator(badger.DefaultIteratorOptions)
	defer it.Close()
	for it.Rewind(); it.Valid(); it.Next() {
		item := it.Item()
		if !strings.HasSuffix(string(item.Key()), txID) {
			continue
		}
		record := &MovementRecord{}
		err := item.Value(func(val []byte) error {
			if err := json.Unmarshal(val, record); err != nil {
				return errors.Wrapf(err, "could not unmarshal key %s", string(item.Key()))
			}
			return nil
		})
		if err != nil {
			return errors.Wrapf(err, "could not get value for key %s", string(item.Key()))
		}
		record.Record.Status = status
		bytes, err := json.Marshal(record)
		if err != nil {
			return errors.Wrapf(err, "could not marshal record for key %s", string(item.Key()))
		}

		err = db.txn.Set(item.Key(), bytes)
		if err != nil {
			return errors.Wrapf(err, "could not set value for key %s", string(item.Key()))
		}
	}
	return nil
}

func (db *Persistence) QueryMovements(enrollmentIDs []string, tokenTypes []string, txStatuses []driver.TxStatus, searchDirection driver.SearchDirection, movementDirection driver.MovementDirection, numRecords int) ([]*driver.MovementRecord, error) {
	txn := db.db.NewTransaction(false)
	it := txn.NewIterator(badger.DefaultIteratorOptions)
	var records RecordSlice
	defer it.Close()

	for it.Rewind(); it.Valid(); it.Next() {
		item := it.Item()
		if !strings.HasPrefix(string(item.Key()), "mv") {
			continue
		}
		record := &MovementRecord{}
		err := item.Value(func(val []byte) error {
			if err := json.Unmarshal(val, record); err != nil {
				return errors.Wrapf(err, "could not unmarshal key %s", string(item.Key()))
			}
			return nil
		})
		if err != nil {
			return nil, errors.Wrapf(err, "could not get movementDirection for key %s", string(item.Key()))
		}

		// filter
		if len(enrollmentIDs) != 0 {
			found := false
			for _, id := range enrollmentIDs {
				if record.Record.EnrollmentID == id {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		if len(tokenTypes) != 0 {
			found := false
			for _, typ := range tokenTypes {
				if record.Record.TokenType == typ {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		if len(txStatuses) != 0 {
			found := false
			for _, st := range txStatuses {
				if record.Record.Status == st {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		} else {
			// exclude the deleted
			if record.Record.Status == driver.Deleted {
				continue
			}
		}

		if movementDirection == driver.Sent && record.Record.Amount.Sign() > 0 {
			continue
		}

		if movementDirection == driver.Received && record.Record.Amount.Sign() < 0 {
			continue
		}

		records = append(records, record)
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

func dbKey(namespace, key string) string {
	return namespace + keys.NamespaceSeparator + key
}

type RecordSlice []*MovementRecord

func (p RecordSlice) Len() int           { return len(p) }
func (p RecordSlice) Less(i, j int) bool { return p[i].Id < p[j].Id }
func (p RecordSlice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

type TransactionIterator struct {
	it   *badger.Iterator
	from *time.Time
	to   *time.Time
}

func (t *TransactionIterator) Close() {
	t.it.Close()
}

func (t *TransactionIterator) Next() (*driver.TransactionRecord, error) {
	for {
		t.it.Next()
		if !t.it.Valid() {
			return nil, nil
		}
		item := t.it.Item()
		if item == nil {
			return nil, nil
		}
		if !strings.HasPrefix(string(item.Key()), "tx") {
			continue
		}
		record := &TransactionRecord{}
		err := item.Value(func(val []byte) error {
			if err := json.Unmarshal(val, record); err != nil {
				return errors.Wrapf(err, "could not unmarshal key %s", string(item.Key()))
			}
			return nil
		})
		if err != nil {
			return nil, errors.Wrapf(err, "could not get transaction for key %s", string(item.Key()))
		}
		// is record in the time range
		if t.from != nil && record.Record.Timestamp.Before(*t.from) {
			continue
		}
		if t.to != nil && record.Record.Timestamp.After(*t.to) {
			return nil, nil
		}
		return record.Record, nil
	}
}
