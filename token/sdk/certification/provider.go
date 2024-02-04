/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package certification

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokendb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/certification"
	kvs2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/certification/kvs"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type KVSStorageProvider struct {
	kvs *kvs.KVS
}

func NewKVSStorageProvider(kvs *kvs.KVS) *KVSStorageProvider {
	return &KVSStorageProvider{kvs: kvs}
}

func (s *KVSStorageProvider) NewStorage(tmsID token2.TMSID) (certification.Storage, error) {
	return kvs2.NewCertificationStorage(s.kvs, tmsID), nil
}

type DBProvider interface {
	DBByTMSId(id token2.TMSID) (*tokendb.DB, error)
}

type DBStorageProvider struct {
	ttxdbProvider DBProvider
}

func NewDBStorageProvider(dbProvider DBProvider) *DBStorageProvider {
	return &DBStorageProvider{ttxdbProvider: dbProvider}
}

func (s *DBStorageProvider) NewStorage(tmsID token2.TMSID) (certification.Storage, error) {
	db, err := s.ttxdbProvider.DBByTMSId(tmsID)
	if err != nil {
		return nil, err
	}
	return &DBCertificationStorage{DB: db}, nil
}

type DBCertificationStorage struct {
	*tokendb.DB
}

func (t *DBCertificationStorage) Exists(id *token.ID) bool {
	return t.DB.ExistsCertification(id)
}

func (t *DBCertificationStorage) Store(certifications map[*token.ID][]byte) error {
	return t.DB.StoreCertifications(certifications)
}
