/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tokenlockdb

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
)

type Manager = db.Manager[*StoreService]

func NewManager(dh *db.DriverHolder) *Manager {
	return db.MappedManager[driver.TokenLockStore, *StoreService](dh.NewTokenLockManager(), newStoreService)
}

type StoreService struct{ driver.TokenLockStore }

func newStoreService(p driver.TokenLockStore) (*StoreService, error) {
	return &StoreService{TokenLockStore: p}, nil
}
