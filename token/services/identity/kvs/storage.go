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

type WalletDB struct {
	kvs   KVS
	tmsID token.TMSID
}

func NewWalletDB(kvs KVS, tmsID token.TMSID) *WalletDB {
	return &WalletDB{kvs: kvs, tmsID: tmsID}
}

func (s *WalletDB) StoreWalletID(wID driver.WalletID) error {
	return s.kvs.Put(kvs.CreateCompositeKeyOrPanic("wallets", []string{s.tmsID.Network, s.tmsID.Channel, s.tmsID.Namespace, wID}), wID)
}

func (s *WalletDB) GetWalletID(id view.Identity) (driver.WalletID, error) {
	var wID driver.WalletID
	if err := s.kvs.Get(id.Hash(), &wID); err != nil {
		return "", err
	}
	return wID, nil
}

func (s *WalletDB) GetWalletIDs(int) ([]driver.WalletID, error) {
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

func (s *WalletDB) StoreIdentity(identity view.Identity, wID driver.WalletID, roleID int, meta any) error {
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

func (s *WalletDB) LoadMeta(identity view.Identity, meta any) error {
	return s.kvs.Get("meta"+identity.Hash(), meta)
}

func (s *WalletDB) IdentityExists(identity view.Identity, wID driver.WalletID) bool {
	return s.kvs.Exists(s.walletPrefix(wID) + identity.Hash())
}

func (s *WalletDB) walletPrefix(wID driver.WalletID) string {
	return fmt.Sprintf("%s-%s-%s-%s", s.tmsID.Network, s.tmsID.Channel, s.tmsID.Namespace, wID)
}

type IdentityDB struct {
	kvs   KVS
	tmsID token.TMSID
}

func NewIdentityDB(kvs KVS, tmsID token.TMSID) *IdentityDB {
	return &IdentityDB{kvs: kvs, tmsID: tmsID}
}

func (s *IdentityDB) AddConfiguration(wp driver.IdentityConfiguration) error {
	k, err := kvs.CreateCompositeKey("token-sdk", []string{"msp", s.tmsID.String(), "registeredIdentity", wp.Type, wp.ID})
	if err != nil {
		return errors.Wrapf(err, "failed to create identity key")
	}
	return s.kvs.Put(k, wp.URL)
}

func (s *IdentityDB) IteratorConfigurations(configurationType string) (driver.Iterator[driver.IdentityConfiguration], error) {
	it, err := s.kvs.GetByPartialCompositeID("token-sdk", []string{"msp", s.tmsID.String(), "registeredIdentity", configurationType})
	if err != nil {
		return nil, errors.WithMessage(err, "failed to get registered identities from kvs")
	}
	return &IdentityConfigurationsIterator{Iterator: it}, nil
}

func (s *IdentityDB) StoreAuditInfo(identity, info []byte) error {
	k := kvs.CreateCompositeKeyOrPanic(
		"fsc.platform.view.sig",
		[]string{
			view.Identity(identity).String(),
		},
	)
	if err := s.kvs.Put(k, info); err != nil {
		return err
	}
	return nil
}

func (s *IdentityDB) GetAuditInfo(identity []byte) ([]byte, error) {
	k := kvs.CreateCompositeKeyOrPanic(
		"fsc.platform.view.sig",
		[]string{
			view.Identity(identity).String(),
		},
	)
	if !s.kvs.Exists(k) {
		return nil, nil
	}
	var res []byte
	if err := s.kvs.Get(k, &res); err != nil {
		return nil, err
	}
	return res, nil
}

func (s *IdentityDB) StoreSignerInfo(id, info []byte) error {
	idHash := view.Identity(id).UniqueID()
	k, err := kvs.CreateCompositeKey("sigService", []string{"signer", idHash})
	if err != nil {
		return errors.Wrap(err, "failed to create composite key to store entry in kvs")
	}
	err = s.kvs.Put(k, info)
	if err != nil {
		return errors.Wrap(err, "failed to store entry in kvs for the passed signer")
	}
	return nil
}

func (s *IdentityDB) SignerInfoExists(id []byte) (bool, error) {
	idHash := view.Identity(id).UniqueID()
	k, err := kvs.CreateCompositeKey("sigService", []string{"signer", idHash})
	if err != nil {
		return false, err
	}
	if s.kvs.Exists(k) {
		return true, nil
	}
	return false, nil
}

type IdentityConfigurationsIterator struct {
	kvs.Iterator
}

func (w *IdentityConfigurationsIterator) Next() (driver.IdentityConfiguration, error) {
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
