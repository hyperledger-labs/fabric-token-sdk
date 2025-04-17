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
	return db.MappedManager[driver.TokenLockDB, *DB](dh.NewTokenLockManager(), newDB)
}

type DB struct{ driver.TokenLockDB }

func newDB(p driver.TokenLockDB) (*DB, error) { return &DB{TokenLockDB: p}, nil }
