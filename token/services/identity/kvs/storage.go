/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package kvs

import (
	"fmt"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/pkg/errors"
)

type KVS interface {
	Exists(id string) bool
	Put(id string, state interface{}) error
	Get(id string, state interface{}) error
	GetByPartialCompositeID(prefix string, attrs []string) (kvs.Iterator, error)
}

type IdentityStorage struct {
	kvs   KVS
	tmsID token.TMSID
}

func NewIdentityStorage(kvs KVS, tmsID token.TMSID) *IdentityStorage {
	return &IdentityStorage{kvs: kvs, tmsID: tmsID}
}

func (s *IdentityStorage) StoreWalletID(wID driver.WalletID) error {
	return s.kvs.Put(kvs.CreateCompositeKeyOrPanic("wallets", []string{s.tmsID.Network, s.tmsID.Channel, s.tmsID.Namespace, wID}), wID)
}

func (s *IdentityStorage) GetWalletID(id view.Identity) (driver.WalletID, error) {
	var wID driver.WalletID
	if err := s.kvs.Get(id.Hash(), &wID); err != nil {
		return "", err
	}
	return wID, nil
}

func (s *IdentityStorage) GetWalletIDs() ([]driver.WalletID, error) {
	it, err := s.kvs.GetByPartialCompositeID("wallets", []string{s.tmsID.Network, s.tmsID.Channel, s.tmsID.Namespace})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get wallets iterator")
	}
	var walletIDs []string
	for it.HasNext() {
		var wID string
		if _, err := it.Next(&wID); err != nil {
			return nil, errors.Wrapf(err, "failed to get next wallets from iterator")
		}
		walletIDs = append(walletIDs, wID)
	}
	return walletIDs, nil
}

func (s *IdentityStorage) StoreIdentity(identity view.Identity, wID driver.WalletID, meta any) error {
	idHash := identity.Hash()
	if err := s.kvs.Put(idHash, wID); err != nil {
		return errors.WithMessagef(err, "failed to store identity's wallet [%s]", identity)
	}
	if meta != nil {
		if err := s.kvs.Put("meta"+idHash, meta); err != nil {
			return errors.WithMessagef(err, "failed to store identity's metadata [%s]", identity)
		}
	}
	if err := s.kvs.Put(s.walletPrefix(wID)+idHash, wID); err != nil {
		return errors.WithMessagef(err, "failed to store identity's wallet reference[%s]", identity)
	}
	return nil
}

func (s *IdentityStorage) LoadMeta(identity view.Identity, meta any) error {
	return s.kvs.Get("meta"+identity.Hash(), meta)
}

func (s *IdentityStorage) IdentityExists(identity view.Identity, wID driver.WalletID) bool {
	return s.kvs.Exists(s.walletPrefix(wID) + identity.Hash())
}

func (s *IdentityStorage) walletPrefix(wID driver.WalletID) string {
	return fmt.Sprintf("%s-%s-%s-%s", s.tmsID.Network, s.tmsID.Channel, s.tmsID.Namespace, wID)
}

type WalletPathStorage struct {
	kvs    KVS
	prefix string
}

func NewWalletPathStorage(kvs KVS, prefix string) *WalletPathStorage {
	return &WalletPathStorage{kvs: kvs, prefix: prefix}
}

func (w *WalletPathStorage) AddConfiguration(wp driver.IdentityConfiguration) error {
	k, err := kvs.CreateCompositeKey("token-sdk", []string{"msp", w.prefix, "registeredIdentity", wp.ID})
	if err != nil {
		return errors.Wrapf(err, "failed to create identity key")
	}
	return w.kvs.Put(k, wp.URL)
}

func (w *WalletPathStorage) IteratorConfigurations() (driver.Iterator[driver.IdentityConfiguration], error) {
	it, err := w.kvs.GetByPartialCompositeID("token-sdk", []string{"msp", w.prefix, "registeredIdentity"})
	if err != nil {
		return nil, errors.WithMessage(err, "failed to get registered identities from kvs")
	}
	return &WalletPathStorageIterator{Iterator: it}, nil
}

type WalletPathStorageIterator struct {
	kvs.Iterator
}

func (w *WalletPathStorageIterator) Next() (driver.IdentityConfiguration, error) {
	var path string
	k, err := w.Iterator.Next(&path)
	if err != nil {
		return driver.IdentityConfiguration{}, err
	}
	_, attrs, err := kvs.SplitCompositeKey(k)
	if err != nil {
		return driver.IdentityConfiguration{}, errors.WithMessagef(err, "failed to split key [%s]", k)
	}
	return driver.IdentityConfiguration{
		ID:  attrs[3],
		URL: path,
	}, nil
}
