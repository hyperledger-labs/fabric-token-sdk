/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package kvs

import (
	"fmt"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
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

func (s *IdentityStorage) StoreWalletID(wID identity.WalletID) error {
	return s.kvs.Put(kvs.CreateCompositeKeyOrPanic("wallets", []string{s.tmsID.Network, s.tmsID.Channel, s.tmsID.Namespace, wID}), wID)
}

func (s *IdentityStorage) GetWalletID(id view.Identity) (identity.WalletID, error) {
	var wID identity.WalletID
	if err := s.kvs.Get(id.Hash(), &wID); err != nil {
		return "", err
	}
	return wID, nil
}

func (s *IdentityStorage) GetWalletIDs() ([]identity.WalletID, error) {
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

func (s *IdentityStorage) StoreIdentity(identity view.Identity, wID identity.WalletID, meta any) error {
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

func (s *IdentityStorage) IdentityExists(identity view.Identity, wID identity.WalletID) bool {
	return s.kvs.Exists(s.walletPrefix(wID) + identity.Hash())
}

func (s *IdentityStorage) walletPrefix(wID identity.WalletID) string {
	return fmt.Sprintf("%s-%s-%s-%s", s.tmsID.Network, s.tmsID.Channel, s.tmsID.Namespace, wID)
}
