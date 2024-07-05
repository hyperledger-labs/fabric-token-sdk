/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package memory

import (
	"database/sql"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
	dbdriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	sqldb "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql"
	sql2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/tokendb/db/sql"
	_ "modernc.org/sqlite"
)

func NewDriver() db.NamedDriver[dbdriver.TokenDBDriver] {
	return db.NamedDriver[dbdriver.TokenDBDriver]{
		Name: "memory",
		Driver: db.NewMemoryDriver(sql2.NewSQLDBOpener(), func(db *sql.DB, tablePrefix string, createSchema bool) (dbdriver.TokenDB, error) {
			return sqldb.NewTokenDB(db, tablePrefix, createSchema)
		}),
	}
}
