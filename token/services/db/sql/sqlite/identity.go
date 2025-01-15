/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sqlite

import (
	"database/sql"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/sqlite"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
)

func NewCachedIdentityDB(k common.Opts) (driver.IdentityDB, error) {
	db, err := sqlite.OpenDB(k.DataSource, k.MaxOpenConns, k.MaxIdleConns, k.MaxIdleTime, k.SkipPragmas)
	if err != nil {
		return nil, err
	}
	return NewIdentityDB(db, common.NewDBOptsFromOpts(k))
}

func NewIdentityDB(db *sql.DB, opts common.NewDBOpts) (driver.IdentityDB, error) {
	return common.NewCachedIdentityDB(db, opts, sqlite.NewInterpreter())
}
