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

	"github.com/dgraph-io/badger/v3"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditor/auditdb/db/badger/keys"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditor/auditdb/driver"
	"github.com/pkg/errors"
)

type Record struct {
	Id     uint64
	Record *driver.Record
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

func (db *Persistence) AddRecord(record *driver.Record) error {
	next, err := db.seq.Next()
	if err != nil {
		return errors.Wrapf(err, "failed getting next index")
	}
	dbKey := dbKey("default", fmt.Sprintf("%d", next))

	value := &Record{
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

func (db *Persistence) SetStatus(txID string, status driver.Status) error {
	panic("implement me")
}

func (db *Persistence) Query(ids []string, types []string, status []driver.Status, direction driver.Direction, value driver.Value, numRecords int) ([]*driver.Record, error) {
	txn := db.db.NewTransaction(false)
	it := txn.NewIterator(badger.DefaultIteratorOptions)
	var records RecordSlice

	for it.Rewind(); it.Valid(); it.Next() {
		item := it.Item()
		if !strings.HasPrefix(string(item.Key()), "default") {
			continue
		}
		record := &Record{}
		err := item.Value(func(val []byte) error {
			if err := json.Unmarshal(val, record); err != nil {
				return errors.Wrapf(err, "could not unmarshal key %s", string(item.Key()))
			}
			return nil
		})
		if err != nil {
			return nil, errors.Wrapf(err, "could not get value for key %s", string(item.Key()))
		}

		// filter
		if len(ids) != 0 {
			found := false
			for _, id := range ids {
				if record.Record.EnrollmentID == id {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		if len(types) != 0 {
			found := false
			for _, typ := range types {
				if record.Record.Type == typ {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		if len(status) != 0 {
			found := false
			for _, st := range status {
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

		if value == driver.Sent && record.Record.Amount.Sign() > 0 {
			continue
		}

		if value == driver.Received && record.Record.Amount.Sign() < 0 {
			continue
		}

		records = append(records, record)
	}

	// Sort
	switch direction {
	case driver.FromBeginning:
		sort.Sort(records)
	case driver.FromLast:
		sort.Sort(sort.Reverse(records))
	}

	if numRecords > 0 && len(records) > numRecords {
		records = records[:numRecords]
	}

	var res []*driver.Record
	for _, record := range records {
		res = append(res, record.Record)
	}

	return res, nil
}

func dbKey(namespace, key string) string {
	return namespace + keys.NamespaceSeparator + key
}

type RecordSlice []*Record

func (p RecordSlice) Len() int           { return len(p) }
func (p RecordSlice) Less(i, j int) bool { return p[i].Id < p[j].Id }
func (p RecordSlice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
