/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package memory

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	sqldb "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb/db/sql"
	_ "modernc.org/sqlite"
)

func NewDriver() db.NamedDriver[driver.TTXDBDriver] {
	return db.NamedDriver[driver.TTXDBDriver]{
		Name:   "memory",
		Driver: db.NewMemoryDriver(sql.NewSQLDBOpener(), sqldb.NewTransactionDB),
	}
}
