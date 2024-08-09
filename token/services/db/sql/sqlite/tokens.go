/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sqlite

import (
	"database/sql"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
)

func NewTokenDB(db *sql.DB, opts common.NewDBOpts) (driver.TokenDB, error) {
	return common.NewTokenDB(db, opts)
}

func NewTokenNDB(db *sql.DB, opts common.NewDBOpts) (driver.TokenNDB, error) {
	return common.NewTokenNDB(db, opts)
}
