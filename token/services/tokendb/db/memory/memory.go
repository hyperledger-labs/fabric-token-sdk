/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package memory

import (
	sqldb "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/drivers"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokendb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokendb/db/sql"
	_ "modernc.org/sqlite"
)

func init() {
	tokendb.Register("memory", drivers.NewMemoryDriver(sql.NewSQLDBOpener(), sqldb.NewTokenDB))
}
