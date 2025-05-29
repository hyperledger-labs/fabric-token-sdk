/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package kvs

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	driver3 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/pkg/errors"
)

const (
	IdentityDBPrefix              = "idb"
	IdentityDBConfigurationPrefix = "configuration"
	IdentityDBData                = "data"
	IdentityDBSigner              = "signer"
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

type IdentityStore struct {
	kvs   KVS
	tmsID token.TMSID
}

func NewIdentityStore(kvs KVS, tmsID token.TMSID) *IdentityStore {
	return &IdentityStore{kvs: kvs, tmsID: tmsID}
}

func (s *IdentityStore) AddConfiguration(wp driver.IdentityConfiguration) error {
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
	return s.kvs.Put(context.Background(), k, &wp)
}

func (s *IdentityStore) IteratorConfigurations(configurationType string) (driver3.IdentityConfigurationIterator, error) {
	it, err := s.kvs.GetByPartialCompositeID(
		context.Background(),
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

func (s *IdentityStore) ConfigurationExists(id, configurationType, url string) (bool, error) {
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
	return s.kvs.Exists(context.Background(), k), nil
}

func (s *IdentityStore) StoreIdentityData(id []byte, identityAudit []byte, tokenMetadata []byte, tokenMetadataAudit []byte) error {
	k := kvs.CreateCompositeKeyOrPanic(
		IdentityDBPrefix,
		[]string{
			IdentityDBData,
			driver2.Identity(id).String(),
		},
	)
	if err := s.kvs.Put(context.Background(), k, &RecipientData{
		AuditInfo:              identityAudit,
		TokenMetadata:          tokenMetadata,
		TokenMetadataAuditInfo: tokenMetadataAudit,
	}); err != nil {
		return err
	}
	return nil
}

func (s *IdentityStore) GetAuditInfo(identity []byte) ([]byte, error) {
	k := kvs.CreateCompositeKeyOrPanic(
		IdentityDBPrefix,
		[]string{
			IdentityDBData,
			driver2.Identity(identity).String(),
		},
	)
	if !s.kvs.Exists(context.Background(), k) {
		return nil, nil
	}
	var res RecipientData
	if err := s.kvs.Get(context.Background(), k, &res); err != nil {
		return nil, err
	}
	return res.AuditInfo, nil
}

func (s *IdentityStore) GetTokenInfo(identity []byte) ([]byte, []byte, error) {
	k := kvs.CreateCompositeKeyOrPanic(
		IdentityDBPrefix,
		[]string{
			IdentityDBData,
			driver2.Identity(identity).String(),
		},
	)
	if !s.kvs.Exists(context.Background(), k) {
		return nil, nil, nil
	}
	var res RecipientData
	if err := s.kvs.Get(context.Background(), k, &res); err != nil {
		return nil, nil, err
	}
	return res.TokenMetadata, res.TokenMetadataAuditInfo, nil
}

func (s *IdentityStore) StoreSignerInfo(id, info []byte) error {
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
	err = s.kvs.Put(context.Background(), k, info)
	if err != nil {
		return errors.Wrap(err, "failed to store entry in kvs for the passed signer")
	}
	return nil
}

func (s *IdentityStore) GetExistingSignerInfo(identities ...driver2.Identity) ([]string, error) {
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
	return s.kvs.GetExisting(context.Background(), keys...), nil
}

func (s *IdentityStore) SignerInfoExists(id []byte) (bool, error) {
	existing, err := s.GetExistingSignerInfo(id)
	if err != nil {
		return false, err
	}
	return len(existing) > 0, nil
}

func (s *IdentityStore) GetSignerInfo(identity []byte) ([]byte, error) {
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
	if err := s.kvs.Get(context.Background(), k, &res); err != nil {
		return nil, err
	}
	return res, nil
}

func (s *IdentityStore) Close() error {
	return nil
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
