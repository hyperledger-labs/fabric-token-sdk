/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package kvs

import (
	"strconv"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/pkg/errors"
)

type WalletStore struct {
	kvs   KVS
	tmsID token.TMSID
}

func NewWalletStore(kvs KVS, tmsID token.TMSID) *WalletStore {
	return &WalletStore{kvs: kvs, tmsID: tmsID}
}

func (s *WalletStore) StoreIdentity(identity driver2.Identity, eID string, wID driver.WalletID, roleID int, meta []byte) error {
	idHash := identity.UniqueID()
	if meta != nil {
		k, err := kvs.CreateCompositeKey("walletDB", []string{s.tmsID.String(), strconv.Itoa(roleID), idHash, wID, "meta"})
		if err != nil {
			return errors.Wrapf(err, "failed to create key")
		}
		if err := s.kvs.Put(k, meta); err != nil {
			return errors.WithMessagef(err, "failed to store identity's metadata [%s]", identity)
		}
	}
	k, err := kvs.CreateCompositeKey("walletDB", []string{s.tmsID.String(), strconv.Itoa(roleID), idHash, wID})
	if err != nil {
		return errors.Wrapf(err, "failed to create key")
	}
	if err := s.kvs.Put(k, wID); err != nil {
		return errors.WithMessagef(err, "failed to store identity's wallet reference[%s]", identity)
	}

	k, err = kvs.CreateCompositeKey("walletDB", []string{s.tmsID.String(), strconv.Itoa(roleID), idHash})
	if err != nil {
		return errors.Wrapf(err, "failed to create key")
	}
	if err := s.kvs.Put(k, wID); err != nil {
		return errors.WithMessagef(err, "failed to store identity's wallet reference[%s]", identity)
	}
	return nil
}

func (s *WalletStore) IdentityExists(identity driver2.Identity, wID driver.WalletID, roleID int) bool {
	idHash := identity.UniqueID()
	k, err := kvs.CreateCompositeKey("walletDB", []string{s.tmsID.String(), strconv.Itoa(roleID), idHash, wID})
	if err != nil {
		return false
	}
	return s.kvs.Exists(k)
}

func (s *WalletStore) GetWalletID(identity driver2.Identity, roleID int) (driver.WalletID, error) {
	idHash := identity.UniqueID()
	k, err := kvs.CreateCompositeKey("walletDB", []string{s.tmsID.String(), strconv.Itoa(roleID), idHash})
	if err != nil {
		return "", errors.Wrapf(err, "failed to create key")
	}
	var wID driver.WalletID
	if err := s.kvs.Get(k, &wID); err != nil {
		return "", err
	}
	return wID, nil
}

func (s *WalletStore) GetWalletIDs(roleID int) ([]driver.WalletID, error) {
	it, err := s.kvs.GetByPartialCompositeID("walletDB", []string{s.tmsID.String(), strconv.Itoa(roleID)})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get wallets iterator")
	}
	walletIDs := collections.NewSet[string]()
	for it.HasNext() {
		var wID string
		if _, err := it.Next(&wID); err != nil {
			return nil, errors.Wrapf(err, "failed to get next wallets from iterator")
		}
		if !walletIDs.Contains(wID) {
			walletIDs.Add(wID)
		}
	}
	return walletIDs.ToSlice(), nil
}

func (s *WalletStore) LoadMeta(identity driver2.Identity, wID driver.WalletID, roleID int) ([]byte, error) {
	idHash := identity.UniqueID()
	k, err := kvs.CreateCompositeKey("walletDB", []string{s.tmsID.String(), strconv.Itoa(roleID), idHash, wID, "meta"})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create key")
	}
	var meta []byte
	if err := s.kvs.Get(k, &meta); err != nil {
		return nil, err
	}
	return meta, nil
}

func (s *WalletStore) Close() error {
	return nil
}
