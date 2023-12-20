/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package backend

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

type Storage struct {
	prefix string
	kvs    KVS
}

func NewStorage(prefix string, kvs KVS) *Storage {
	return &Storage{prefix: prefix, kvs: kvs}
}

func (s *Storage) Put(tmsID token.TMSID, walletID string) error {
	id := tmsID.String() + walletID
	if err := s.kvs.Put(kvs.CreateCompositeKeyOrPanic(s.prefix, []string{id}), &storage.DBEntry{
		TMSID:    tmsID,
		WalletID: walletID,
	}); err != nil {
		return errors.Wrapf(err, "failed to store db entry in KVS [%s]", tmsID)
	}
	return nil
}

func (s *Storage) Iterator() (storage.Iterator[*storage.DBEntry], error) {
	it, err := s.kvs.GetByPartialCompositeID(s.prefix, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get db list iterator")
	}

	return &iterator{Iterator: it}, nil
}

type iterator struct {
	kvs.Iterator
}

func (i *iterator) Next() (*storage.DBEntry, error) {
	if !i.Iterator.HasNext() {
		return nil, nil
	}
	e := &storage.DBEntry{}
	if _, err := i.Iterator.Next(e); err != nil {
		return nil, errors.Wrapf(err, "failed to get entry")
	}
	return e, nil
}

func (i *iterator) Close() error {
	return i.Iterator.Close()
}
