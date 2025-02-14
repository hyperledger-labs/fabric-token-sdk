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

func NewAuditTransactionDB(readDB, writeDB *sql.DB, opts common.NewDBOpts) (driver.AuditTransactionDB, error) {
	return common.NewAuditTransactionDB(readDB, writeDB, opts, common.NewTokenInterpreter(sqlite.NewInterpreter()))
}

func NewTransactionDB(readDB, writeDB *sql.DB, opts common.NewDBOpts) (driver.TokenTransactionDB, error) {
	return common.NewTransactionDB(readDB, writeDB, opts, common.NewTokenInterpreter(sqlite.NewInterpreter()))
}
