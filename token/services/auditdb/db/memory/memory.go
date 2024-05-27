/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package memory

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditdb/db/sql"
	sqldb "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/drivers"
	_ "modernc.org/sqlite"
)

func init() {
	auditdb.Register("memory", drivers.NewMemoryDriver(sql.NewSQLDBOpener(), sqldb.NewAuditTransactionDB)) //TODO NewTransactionDB
}
