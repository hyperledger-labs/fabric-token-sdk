/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package certification

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
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

type TTXDBProvider interface {
	DBByTMSId(id token2.TMSID) (*ttxdb.DB, error)
}

type TTXDBStorageProvider struct {
	ttxdbProvider TTXDBProvider
}

func NewTTXDBStorageProvider(ttxdbProvider TTXDBProvider) *TTXDBStorageProvider {
	return &TTXDBStorageProvider{ttxdbProvider: ttxdbProvider}
}

func (s *TTXDBStorageProvider) NewStorage(tmsID token2.TMSID) (certification.Storage, error) {
	db, err := s.ttxdbProvider.DBByTMSId(tmsID)
	if err != nil {
		return nil, err
	}
	return &TTXDBCertificationStorage{DB: db}, nil
}

type TTXDBCertificationStorage struct {
	*ttxdb.DB
}

func (t *TTXDBCertificationStorage) Exists(id *token.ID) bool {
	return t.DB.ExistsCertification(id)
}

func (t *TTXDBCertificationStorage) Store(certifications map[*token.ID][]byte) error {
	return t.DB.StoreCertifications(certifications)
}
