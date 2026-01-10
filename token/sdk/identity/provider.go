/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identity

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/identitydb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/keystoredb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/walletdb"
)

type DBStorageProvider struct {
	identityStoreServiceManager identitydb.StoreServiceManager
	walletStoreServiceManager   walletdb.StoreServiceManager
	keyStoreStoreServiceManager keystoredb.StoreServiceManager
}

func NewDBStorageProvider(
	identityStoreServiceManager identitydb.StoreServiceManager,
	walletStoreServiceManager walletdb.StoreServiceManager,
	keyStoreStoreServiceManager keystoredb.StoreServiceManager,
) *DBStorageProvider {
	return &DBStorageProvider{
		identityStoreServiceManager: identityStoreServiceManager,
		walletStoreServiceManager:   walletStoreServiceManager,
		keyStoreStoreServiceManager: keyStoreStoreServiceManager,
	}
}

func (s *DBStorageProvider) WalletStore(tmsID token.TMSID) (driver.WalletStoreService, error) {
	return s.walletStoreServiceManager.StoreServiceByTMSId(tmsID)
}

func (s *DBStorageProvider) IdentityStore(tmsID token.TMSID) (driver.IdentityStoreService, error) {
	return s.identityStoreServiceManager.StoreServiceByTMSId(tmsID)
}

func (s *DBStorageProvider) Keystore(tmsID token.TMSID) (driver.Keystore, error) {
	return s.keyStoreStoreServiceManager.StoreServiceByTMSId(tmsID)
}
