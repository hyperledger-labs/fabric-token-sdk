/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identity

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/identitydb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/walletdb"
)

type StorageProvider struct {
	kvs                         driver.Keystore
	identityStoreServiceManager identitydb.StoreServiceManager
	walletStoreServiceManager   walletdb.StoreServiceManager
}

func NewStorageProvider(
	kvs driver.Keystore,
	identityStoreServiceManager identitydb.StoreServiceManager,
	walletStoreServiceManager walletdb.StoreServiceManager,
) *StorageProvider {
	return &StorageProvider{
		kvs:                         kvs,
		identityStoreServiceManager: identityStoreServiceManager,
		walletStoreServiceManager:   walletStoreServiceManager,
	}
}

func (s *StorageProvider) WalletStore(tmsID token.TMSID) (driver.WalletStoreService, error) {
	return s.walletStoreServiceManager.StoreServiceByTMSId(tmsID)
}

func (s *StorageProvider) IdentityStore(tmsID token.TMSID) (driver.IdentityStoreService, error) {
	return s.identityStoreServiceManager.StoreServiceByTMSId(tmsID)
}

func (s *StorageProvider) Keystore() (driver.Keystore, error) {
	return s.kvs, nil
}
