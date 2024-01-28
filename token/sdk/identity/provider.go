/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identity

import (
	"github.com/IBM/idemix/bccsp/keystore"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity/kvs"
)

type KVSStorageProvider struct {
	kvs kvs.KVS
}

func NewKVSStorageProvider(kvs kvs.KVS) *KVSStorageProvider {
	return &KVSStorageProvider{kvs: kvs}
}

func (s *KVSStorageProvider) NewStorage(tmsID token.TMSID) (identity.Storage, error) {
	return kvs.NewIdentityStorage(s.kvs, tmsID), nil
}

func (s *KVSStorageProvider) GetWalletPathStorage(id string) (identity.WalletPathStorage, error) {
	return kvs.NewWalletPathStorage(s.kvs, id), nil
}

func (s *KVSStorageProvider) NewKeystore() (keystore.KVS, error) {
	return s.kvs, nil
}
