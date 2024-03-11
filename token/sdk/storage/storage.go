/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package storage

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage"
	"github.com/pkg/errors"
)

type KVS interface {
	Put(id string, state interface{}) error
	GetByPartialCompositeID(prefix string, attrs []string) (kvs.Iterator, error)
}

type DBEntriesStorage struct {
	prefix string
	kvs    KVS
}

func NewDBEntriesStorage(prefix string, kvs KVS) *DBEntriesStorage {
	return &DBEntriesStorage{prefix: prefix, kvs: kvs}
}

func (s *DBEntriesStorage) Put(tmsID token.TMSID, walletID string) error {
	id := tmsID.String() + walletID
	if err := s.kvs.Put(kvs.CreateCompositeKeyOrPanic(s.prefix, []string{id}), &storage.DBEntry{
		TMSID:    tmsID,
		WalletID: walletID,
	}); err != nil {
		return errors.Wrapf(err, "failed to store db entry in KVS [%s]", tmsID)
	}
	return nil
}

func (s *DBEntriesStorage) Iterator() (storage.Iterator[*storage.DBEntry], error) {
	it, err := s.kvs.GetByPartialCompositeID(s.prefix, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get db list iterator")
	}

	return &DBEntriesStorageIterator{Iterator: it}, nil
}

func (s *DBEntriesStorage) ByTMS(tmsID token.TMSID) ([]*storage.DBEntry, error) {
	itr, err := s.kvs.GetByPartialCompositeID(s.prefix, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get db list iterator")
	}
	it := DBEntriesStorageIterator{Iterator: itr}
	defer it.Close()

	entries := []*storage.DBEntry{}
	for {
		if !it.HasNext() {
			return entries, nil
		}
		entry, err := it.Next()
		if err != nil {
			return entries, errors.Wrapf(err, "failed to get next entry for [%s:%s]...", entry.TMSID, entry.WalletID)
		}
		if entry.TMSID.Equal(tmsID) {
			entries = append(entries, entry)
		}
	}
}

type DBEntriesStorageIterator struct {
	kvs.Iterator
}

func (i *DBEntriesStorageIterator) Next() (*storage.DBEntry, error) {
	e := &storage.DBEntry{}
	if _, err := i.Iterator.Next(e); err != nil {
		return nil, errors.Wrapf(err, "failed to get entry")
	}
	return e, nil
}

func (i *DBEntriesStorageIterator) Close() error {
	return i.Iterator.Close()
}
