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

func NewTokenNotifier(db *sql.DB, opts common.NewDBOpts) (driver.TokenNotifier, error) {
	panic("not implemented")
}
