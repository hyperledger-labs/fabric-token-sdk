/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identity

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/kvs"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identitydb"
)

type KVSStorageProvider struct {
	kvs kvs.KVS
}

func NewKVSStorageProvider(kvs kvs.KVS) *KVSStorageProvider {
	return &KVSStorageProvider{kvs: kvs}
}

func (s *KVSStorageProvider) OpenWalletDB(tmsID token.TMSID) (driver.WalletDB, error) {
	return kvs.NewIdentityStorage(s.kvs, tmsID), nil
}

func (s *KVSStorageProvider) OpenIdentityDB(tmsID token.TMSID, id string) (driver.IdentityDB, error) {
	return kvs.NewWalletPathStorage(s.kvs, tmsID.String()+id), nil
}

func (s *KVSStorageProvider) NewKeystore() (identity.KVS, error) {
	return s.kvs, nil
}

type DBStorageProvider struct {
	kvs     kvs.KVS
	manager *identitydb.Manager
}

func NewDBStorageProvider(kvs kvs.KVS, manager *identitydb.Manager) *DBStorageProvider {
	return &DBStorageProvider{kvs: kvs, manager: manager}
}

func (s *DBStorageProvider) OpenWalletDB(tmsID token.TMSID) (driver.WalletDB, error) {
	return s.manager.WalletDBByTMSId(tmsID)
}

func (s *DBStorageProvider) OpenIdentityDB(tmsID token.TMSID, id identitydb.IdType) (driver.IdentityDB, error) {
	return s.manager.IdentityDBByTMSId(tmsID, id)
}

func (s *DBStorageProvider) NewKeystore() (identity.KVS, error) {
	return s.kvs, nil
}
