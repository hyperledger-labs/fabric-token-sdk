/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package memory

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
	dbdriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	sqldb "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokenlockdb/db/sql"
)

func NewDriver() db.NamedDriver[dbdriver.TokenLockDBDriver] {
	return db.NamedDriver[dbdriver.TokenLockDBDriver]{
		Name:   "memory",
		Driver: db.NewMemoryDriver(sql.NewSQLDBOpener(), sqldb.NewTokenLockDB),
	}
}
