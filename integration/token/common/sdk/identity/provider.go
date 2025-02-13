/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identity

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/storage/kvs"
)

type KVSStorageProvider struct {
	kvs kvs.KVS
}

func NewKVSStorageProvider(kvs kvs.KVS) *KVSStorageProvider {
	return &KVSStorageProvider{kvs: kvs}
}

func (s *KVSStorageProvider) OpenWalletDB(tmsID token.TMSID) (driver.WalletDB, error) {
	return kvs.NewWalletDB(s.kvs, tmsID), nil
}

func (s *KVSStorageProvider) OpenIdentityDB(tmsID token.TMSID) (driver.IdentityDB, error) {
	return kvs.NewIdentityDB(s.kvs, tmsID), nil
}

func (s *KVSStorageProvider) NewKeystore() (identity.Keystore, error) {
	return s.kvs, nil
}
