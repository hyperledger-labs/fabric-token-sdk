/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package memory

import (
	mem "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/memory"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
	dbdriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	sqlite2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/sqlite"
)

func NewDriver() db.NamedDriver[dbdriver.TokenLockDBDriver] {
	return db.NamedDriver[dbdriver.TokenLockDBDriver]{
		Name:   mem.MemoryPersistence,
		Driver: db.NewMemoryDriver[dbdriver.TokenLockDB](sqlite2.NewTokenLockDB),
	}
}
