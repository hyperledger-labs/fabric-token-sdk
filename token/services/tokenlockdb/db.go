/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tokenlockdb

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
)

type (
	Holder  = *db.DriverHolder[*DB, driver.TokenLockDB, driver.TokenLockDBDriver]
	Manager = db.Manager[*DB, driver.TokenLockDB, driver.TokenLockDBDriver]
)

func NewHolder(drivers []db.NamedDriver[driver.TokenLockDBDriver]) Holder {
	return db.NewDriverHolder[*DB, driver.TokenLockDB, driver.TokenLockDBDriver](newDB, drivers...)
}

type DB struct{ driver.TokenLockDB }

func newDB(p driver.TokenLockDB) *DB { return &DB{TokenLockDB: p} }
