/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package memory

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
	sqldb "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokendb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokendb/db/sql"
	_ "modernc.org/sqlite"
)

func init() {
	tokendb.Register("memory", db.NewMemoryDriver(sql.NewSQLDBOpener(), sqldb.NewTokenDB))
}
