/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package walletdb

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/multiplexed"
)

type StoreServiceManager db.StoreServiceManager[*StoreService]

func NewStoreServiceManager(cp db.ConfigService, drivers multiplexed.Driver) StoreServiceManager {
	return db.NewStoreServiceManager(cp, "identitydb.persistence", drivers.NewWallet, newStoreService)
}

// StoreService is a database that stores wallet related information
type StoreService struct {
	driver.WalletStore
}

func newStoreService(p driver.WalletStore) (*StoreService, error) {
	return &StoreService{WalletStore: p}, nil
}
