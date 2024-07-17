/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package memory

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditdb/db/sql"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
	dbdriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	sqldb "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql"
)

func NewDriver() db.NamedDriver[dbdriver.AuditDBDriver] {
	return db.NamedDriver[dbdriver.AuditDBDriver]{
		Name:   "memory",
		Driver: db.NewMemoryDriver(sql.NewSQLDBOpener(), sqldb.NewAuditTransactionDB), // TODO: NewTransactionDB
	}
}
