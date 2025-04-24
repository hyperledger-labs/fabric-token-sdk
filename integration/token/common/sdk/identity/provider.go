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

func (s *KVSStorageProvider) WalletStore(tmsID token.TMSID) (driver.WalletStore, error) {
	return kvs.NewWalletStore(s.kvs, tmsID), nil
}

func (s *KVSStorageProvider) IdentityStore(tmsID token.TMSID) (driver.IdentityStore, error) {
	return kvs.NewIdentityStore(s.kvs, tmsID), nil
}

func (s *KVSStorageProvider) Keystore() (identity.Keystore, error) {
	return s.kvs, nil
}
