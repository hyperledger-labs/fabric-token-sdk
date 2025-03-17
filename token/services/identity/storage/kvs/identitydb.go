/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package kvs

import (
	"encoding/base64"
	"fmt"

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

type IdentityDB struct {
	kvs   KVS
	tmsID token.TMSID
}

func NewIdentityDB(kvs KVS, tmsID token.TMSID) *IdentityDB {
	return &IdentityDB{kvs: kvs, tmsID: tmsID}
}

func (s *IdentityDB) AddConfiguration(wp driver.IdentityConfiguration) error {
	k, err := kvs.CreateCompositeKey(
		IdentityDBPrefix,
		[]string{
			IdentityDBConfigurationPrefix,
			s.tmsID.String(),
			wp.Type,
			mergeIDURL(wp.ID, wp.URL),
		},
	)
	if err != nil {
		return errors.Wrapf(err, "failed to create key")
	}
	return s.kvs.Put(k, &wp)
}

func (s *IdentityDB) IteratorConfigurations(configurationType string) (driver.Iterator[driver.IdentityConfiguration], error) {
	it, err := s.kvs.GetByPartialCompositeID(
		IdentityDBPrefix,
		[]string{
			IdentityDBConfigurationPrefix,
			s.tmsID.String(),
			configurationType,
		},
	)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to get registered identities from kvs")
	}
	return &IdentityConfigurationsIterator{Iterator: it}, nil
}

func (s *IdentityDB) ConfigurationExists(id, configurationType, url string) (bool, error) {
	k, err := kvs.CreateCompositeKey(
		IdentityDBPrefix,
		[]string{
			IdentityDBConfigurationPrefix,
			s.tmsID.String(),
			configurationType,
			mergeIDURL(id, url),
		},
	)
	if err != nil {
		return false, errors.Wrapf(err, "failed to create key")
	}
	return s.kvs.Exists(k), nil
}

func (s *IdentityDB) StoreIdentityData(id []byte, identityAudit []byte, tokenMetadata []byte, tokenMetadataAudit []byte) error {
	k := kvs.CreateCompositeKeyOrPanic(
		IdentityDBPrefix,
		[]string{
			IdentityDBData,
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
		IdentityDBPrefix,
		[]string{
			IdentityDBData,
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
		IdentityDBPrefix,
		[]string{
			IdentityDBData,
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
	return res.TokenMetadata, res.TokenMetadataAuditInfo, nil
}

func (s *IdentityDB) StoreSignerInfo(id, info []byte) error {
	idHash := driver2.Identity(id).UniqueID()
	k, err := kvs.CreateCompositeKey(
		IdentityDBPrefix,
		[]string{
			IdentityDBSigner,
			idHash,
		},
	)
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
		k, err := kvs.CreateCompositeKey(
			IdentityDBPrefix,
			[]string{
				IdentityDBSigner,
				id.UniqueID(),
			},
		)
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
	k, err := kvs.CreateCompositeKey(
		IdentityDBPrefix,
		[]string{
			IdentityDBSigner,
			idHash,
		},
	)
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
	idConfig := &driver.IdentityConfiguration{}
	_, err := w.Iterator.Next(idConfig)
	if err != nil {
		return driver.IdentityConfiguration{}, err
	}
	return *idConfig, nil
}

func mergeIDURL(id string, url string) string {
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s%s", id, url)))
}
