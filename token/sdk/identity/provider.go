/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identity

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identitydb"
)

type DBStorageProvider struct {
	kvs     identity.Keystore
	manager *identitydb.Manager
}

func NewDBStorageProvider(kvs identity.Keystore, manager *identitydb.Manager) *DBStorageProvider {
	return &DBStorageProvider{kvs: kvs, manager: manager}
}

func (s *DBStorageProvider) WalletStore(tmsID token.TMSID) (driver.WalletStore, error) {
	return s.manager.WalletStoreByTMSId(tmsID)
}

func (s *DBStorageProvider) IdentityStore(tmsID token.TMSID) (driver.IdentityStore, error) {
	return s.manager.IdentityStoreByTMSId(tmsID)
}

func (s *DBStorageProvider) Keystore() (identity.Keystore, error) {
	return s.kvs, nil
}
