/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identity

import (
	"github.com/IBM/idemix/bccsp/keystore"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	kvs2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity/kvs"
)

type KVSStorageProvider struct {
	sp view.ServiceProvider
}

func NewKVSStorageProvider(sp view.ServiceProvider) *KVSStorageProvider {
	return &KVSStorageProvider{sp: sp}
}

func (s *KVSStorageProvider) NewStorage(tmsID token.TMSID) (identity.Storage, error) {
	return kvs.NewIdentityStorage(kvs2.GetService(s.sp), tmsID), nil
}

func (s *KVSStorageProvider) GetWalletPathStorage(id string) (identity.WalletPathStorage, error) {
	return kvs.NewWalletPathStorage(kvs2.GetService(s.sp), id), nil
}

func (s *KVSStorageProvider) NewKeystore() (keystore.KVS, error) {
	return kvs2.GetService(s.sp), nil
}
