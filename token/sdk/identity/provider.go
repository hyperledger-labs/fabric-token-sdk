/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identity

import (
	"github.com/IBM/idemix/bccsp/keystore"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/kvs"
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

func (s *KVSStorageProvider) NewKeystore() (keystore.KVS, error) {
	return s.kvs, nil
}
