/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identity

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identitydb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/walletdb"
)

type DBStorageProvider struct {
	kvs                         driver.Keystore
	identityStoreServiceManager identitydb.StoreServiceManager
	walletStoreServiceManager   walletdb.StoreServiceManager
}

func NewDBStorageProvider(
	kvs driver.Keystore,
	identityStoreServiceManager identitydb.StoreServiceManager,
	walletStoreServiceManager walletdb.StoreServiceManager,
) *DBStorageProvider {
	return &DBStorageProvider{
		kvs:                         kvs,
		identityStoreServiceManager: identityStoreServiceManager,
		walletStoreServiceManager:   walletStoreServiceManager,
	}
}

func (s *DBStorageProvider) WalletStore(tmsID token.TMSID) (driver.WalletStoreService, error) {
	return s.walletStoreServiceManager.StoreServiceByTMSId(tmsID)
}

func (s *DBStorageProvider) IdentityStore(tmsID token.TMSID) (driver.IdentityStoreService, error) {
	return s.identityStoreServiceManager.StoreServiceByTMSId(tmsID)
}

func (s *DBStorageProvider) Keystore() (driver.Keystore, error) {
	return s.kvs, nil
}
