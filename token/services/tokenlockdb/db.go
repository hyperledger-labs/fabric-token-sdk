/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tokenlockdb

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
)

type Manager = db.Manager[*DB]

func NewManager(dh *db.DriverHolder) *Manager {
	return db.MappedManager[driver.TokenLockStore, *DB](dh.NewTokenLockManager(), newDB)
}

type DB struct{ driver.TokenLockStore }

func newDB(p driver.TokenLockStore) (*DB, error) { return &DB{TokenLockStore: p}, nil }
