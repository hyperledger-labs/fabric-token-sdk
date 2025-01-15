/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package kvs

import (
	"strconv"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/pkg/errors"
)

// RecipientData contains information about the identity of a token owner
type RecipientData struct {
	// AuditInfo contains private information Identity
	AuditInfo []byte
	// TokenMetadata contains public information related to the token to be assigned to this Recipient.
	TokenMetadata []byte
	// TokenMetadataAuditInfo contains private information TokenMetadata
	TokenMetadataAuditInfo []byte
}

type KVS interface {
	Exists(id string) bool
	GetExisting(ids ...string) []string
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

func (s *WalletDB) StoreIdentity(identity driver2.Identity, eID string, wID driver.WalletID, roleID int, meta []byte) error {
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

func (s *WalletDB) IdentityExists(identity driver2.Identity, wID driver.WalletID, roleID int) bool {
	idHash := identity.UniqueID()
	k, err := kvs.CreateCompositeKey("walletDB", []string{s.tmsID.String(), strconv.Itoa(roleID), idHash, wID})
	if err != nil {
		return false
	}
	return s.kvs.Exists(k)
}

func (s *WalletDB) GetWalletID(identity driver2.Identity, roleID int) (driver.WalletID, error) {
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

func (s *WalletDB) GetWalletIDs(roleID int) ([]driver.WalletID, error) {
	it, err := s.kvs.GetByPartialCompositeID("walletDB", []string{s.tmsID.String(), strconv.Itoa(roleID)})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get wallets iterator")
	}
	var walletIDs []string
	for it.HasNext() {
		var wID string
		if _, err := it.Next(&wID); err != nil {
			return nil, errors.Wrapf(err, "failed to get next wallets from iterator")
		}
		found := false
		for _, walletID := range walletIDs {
			if walletID == wID {
				found = true
			}
		}
		if !found {
			walletIDs = append(walletIDs, wID)
		}
	}
	return walletIDs, nil
}

func (s *WalletDB) LoadMeta(identity driver2.Identity, wID driver.WalletID, roleID int) ([]byte, error) {
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
		return errors.Wrapf(err, "failed to create key")
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

func (s *IdentityDB) ConfigurationExists(id, typ string) (bool, error) {
	k, err := kvs.CreateCompositeKey("token-sdk", []string{"msp", s.tmsID.String(), "registeredIdentity", typ, id})
	if err != nil {
		return false, errors.Wrapf(err, "failed to create key")
	}
	return s.kvs.Exists(k), nil
}

func (s *IdentityDB) StoreIdentityData(id []byte, identityAudit []byte, tokenMetadata []byte, tokenMetadataAudit []byte) error {
	k := kvs.CreateCompositeKeyOrPanic(
		"fsc.platform.view.sig",
		[]string{
			driver2.Identity(id).String(),
		},
	)
	if err := s.kvs.Put(k, &RecipientData{
		AuditInfo:              identityAudit,
		TokenMetadata:          tokenMetadata,
		TokenMetadataAuditInfo: tokenMetadataAudit,
	}); err != nil {
		return err
	}
	return nil
}

func (s *IdentityDB) GetAuditInfo(identity []byte) ([]byte, error) {
	k := kvs.CreateCompositeKeyOrPanic(
		"fsc.platform.view.sig",
		[]string{
			driver2.Identity(identity).String(),
		},
	)
	if !s.kvs.Exists(k) {
		return nil, nil
	}
	var res RecipientData
	if err := s.kvs.Get(k, &res); err != nil {
		return nil, err
	}
	return res.AuditInfo, nil
}

func (s *IdentityDB) GetTokenInfo(identity []byte) ([]byte, []byte, error) {
	k := kvs.CreateCompositeKeyOrPanic(
		"fsc.platform.view.sig",
		[]string{
			driver2.Identity(identity).String(),
		},
	)
	if !s.kvs.Exists(k) {
		return nil, nil, nil
	}
	var res RecipientData
	if err := s.kvs.Get(k, &res); err != nil {
		return nil, nil, err
	}
	return res.TokenMetadata, res.TokenMetadata, nil
}

func (s *IdentityDB) StoreSignerInfo(id, info []byte) error {
	idHash := driver2.Identity(id).UniqueID()
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

func (s *IdentityDB) GetExistingSignerInfo(identities ...driver2.Identity) ([]string, error) {
	keys := make([]string, len(identities))
	for i, id := range identities {
		k, err := kvs.CreateCompositeKey("sigService", []string{"signer", id.UniqueID()})
		if err != nil {
			return nil, err
		}
		keys[i] = k
	}
	return s.kvs.GetExisting(keys...), nil
}

func (s *IdentityDB) SignerInfoExists(id []byte) (bool, error) {
	existing, err := s.GetExistingSignerInfo(id)
	if err != nil {
		return false, err
	}
	return len(existing) > 0, nil
}

func (s *IdentityDB) GetSignerInfo(identity []byte) ([]byte, error) {
	idHash := driver2.Identity(identity).UniqueID()
	k, err := kvs.CreateCompositeKey("sigService", []string{"signer", idHash})
	if err != nil {
		return nil, err
	}
	var res []byte
	if err := s.kvs.Get(k, &res); err != nil {
		return nil, err
	}
	return res, nil
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
