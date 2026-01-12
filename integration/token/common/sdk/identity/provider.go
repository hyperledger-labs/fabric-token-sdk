/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identity

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/storage/kvs"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/driver"
)

type KVSStorageProvider struct {
	kvs kvs.KVS
}

func NewKVSStorageProvider(kvs kvs.KVS) *KVSStorageProvider {
	return &KVSStorageProvider{kvs: kvs}
}

func (s *KVSStorageProvider) WalletStore(tmsID token.TMSID) (driver.WalletStore, error) {
	return kvs.NewWalletStore(s.kvs, tmsID), nil
}

func (s *KVSStorageProvider) IdentityStore(tmsID token.TMSID) (driver.IdentityStore, error) {
	return kvs.NewIdentityStore(s.kvs, tmsID), nil
}

func (s *KVSStorageProvider) Keystore(token.TMSID) (driver2.Keystore, error) {
	return kvs.Keystore(s.kvs), nil
}
