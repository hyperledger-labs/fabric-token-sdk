/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identitydb

import (
	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/multiplexed"
)

type StoreServiceManager db.StoreServiceManager[*StoreService]

func NewStoreServiceManager(cp driver2.ConfigService, drivers multiplexed.Driver) StoreServiceManager {
	return db.NewStoreServiceManager(config.NewService(cp), "identitydb.persistence", drivers.NewIdentity, newStoreService)
}

// StoreService is a database that stores identity related information
type StoreService struct {
	driver.IdentityStore
}

func newStoreService(p driver.IdentityStore) (*StoreService, error) {
	return &StoreService{IdentityStore: p}, nil
}
