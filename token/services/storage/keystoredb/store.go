/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package keystoredb

import (
	cdriver "github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/multiplexed"
)

type StoreServiceManager db.StoreServiceManager[*StoreService]

func NewStoreServiceManager(cp cdriver.ConfigService, drivers multiplexed.Driver) StoreServiceManager {
	return db.NewStoreServiceManager(config.NewService(cp), "keystoredb.persistence", drivers.NewKeyStore, newStoreService)
}

// StoreService is a database that stores identity related information
type StoreService struct {
	driver.KeyStore
}

func newStoreService(p driver.KeyStore) (*StoreService, error) {
	return &StoreService{KeyStore: p}, nil
}
