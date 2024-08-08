/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package memory

import (
	mem "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/memory"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	sqldb "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb/db/sql"
)

func NewDriver() db.NamedDriver[driver.TTXDBDriver] {
	return db.NamedDriver[driver.TTXDBDriver]{
		Name:   mem.MemoryPersistence,
		Driver: db.NewMemoryDriver(sql.NewSQLDBOpener(), sqldb.NewTransactionDB),
	}
}
