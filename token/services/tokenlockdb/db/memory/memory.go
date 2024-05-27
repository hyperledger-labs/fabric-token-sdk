/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package memory

import (
	sqldb "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/drivers"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokenlockdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokenlockdb/db/sql"
	_ "modernc.org/sqlite"
)

func init() {
	tokenlockdb.Register("memory", drivers.NewMemoryDriver(sql.NewSQLDBOpener(), sqldb.NewTokenLockDB))
}
