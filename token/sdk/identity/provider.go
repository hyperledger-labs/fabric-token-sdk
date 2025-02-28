/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identity

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identitydb"
)

type DBStorageProvider struct {
	kvs     identity.Keystore
	manager *identitydb.Manager
}

func NewDBStorageProvider(kvs identity.Keystore, manager *identitydb.Manager) *DBStorageProvider {
	return &DBStorageProvider{kvs: kvs, manager: manager}
}

func (s *DBStorageProvider) WalletDB(tmsID token.TMSID) (driver.WalletDB, error) {
	return s.manager.WalletDBByTMSId(tmsID)
}

func (s *DBStorageProvider) IdentityDB(tmsID token.TMSID) (driver.IdentityDB, error) {
	return s.manager.IdentityDBByTMSId(tmsID)
}

func (s *DBStorageProvider) Keystore() (identity.Keystore, error) {
	return s.kvs, nil
}
