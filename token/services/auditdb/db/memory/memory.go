/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package memory

import (
	mem "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/memory"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditdb/db/sql"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
	dbdriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	sqldb "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql"
)

func NewDriver() db.NamedDriver[dbdriver.AuditDBDriver] {
	return db.NamedDriver[dbdriver.AuditDBDriver]{
		Name:   mem.MemoryPersistence,
		Driver: db.NewMemoryDriver(sql.NewSQLDBOpener(), sqldb.NewAuditTransactionDB), // TODO: NewTransactionDB
	}
}
