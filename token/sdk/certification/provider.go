/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package certification

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/certification"
	kvs2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/certification/kvs"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type KVSStorageProvider struct {
	sp view.ServiceProvider
}

func NewKVSStorageProvider(sp view.ServiceProvider) *KVSStorageProvider {
	return &KVSStorageProvider{sp: sp}
}

func (s *KVSStorageProvider) NewStorage(tmsID token2.TMSID) (certification.Storage, error) {
	return kvs2.NewCertificationStorage(kvs.GetService(s.sp), tmsID), nil
}

type TTXDBStorageProvider struct {
	sp view.ServiceProvider
}

func NewTTXDBStorageProvider(sp view.ServiceProvider) *TTXDBStorageProvider {
	return &TTXDBStorageProvider{sp: sp}
}

func (s *TTXDBStorageProvider) NewStorage(tmsID token2.TMSID) (certification.Storage, error) {
	return &TTXDBCertificationStorage{ttxdb.GetByTMSId(s.sp, tmsID)}, nil
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
