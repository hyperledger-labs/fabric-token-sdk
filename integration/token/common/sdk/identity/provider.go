/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identity

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/storage/kvs"
)

type KVSStorageProvider struct {
	kvs kvs.KVS
}

func NewKVSStorageProvider(kvs kvs.KVS) *KVSStorageProvider {
	return &KVSStorageProvider{kvs: kvs}
}

func (s *KVSStorageProvider) WalletDB(tmsID token.TMSID) (identity.WalletDB, error) {
	return kvs.NewWalletDB(s.kvs, tmsID), nil
}

func (s *KVSStorageProvider) IdentityDB(tmsID token.TMSID) (identity.IdentityDB, error) {
	return kvs.NewIdentityDB(s.kvs, tmsID), nil
}

func (s *KVSStorageProvider) Keystore() (identity.Keystore, error) {
	return s.kvs, nil
}
