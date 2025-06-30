/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tokenlockdb

import (
	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/multiplexed"
)

type StoreServiceManager db.StoreServiceManager[*StoreService]

func NewStoreServiceManager(drivers multiplexed.Driver, cp driver2.ConfigService) StoreServiceManager {
	return db.NewStoreServiceManager(config.NewService(cp), "tokenlockdb.persistence", drivers.NewTokenLock, newStoreService)
}

type StoreService struct{ driver.TokenLockStore }

func newStoreService(p driver.TokenLockStore) (*StoreService, error) {
	return &StoreService{TokenLockStore: p}, nil
}
