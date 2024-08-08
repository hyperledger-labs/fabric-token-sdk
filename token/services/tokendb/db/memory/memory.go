/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package memory

import (
	mem "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/memory"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
	dbdriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	sqldb "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokendb/db/sql"
)

func NewDriver() db.NamedDriver[dbdriver.TokenNDBDriver] {
	return db.NamedDriver[dbdriver.TokenNDBDriver]{
		Name:   mem.MemoryPersistence,
		Driver: db.NewMemoryDriver(sql.NewSQLDBOpener(), sqldb.NewTokenNDB),
	}
}
