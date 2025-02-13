/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package postgres

import (
	"database/sql"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/postgres"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
)

func NewAuditTransactionDB(db *sql.DB, opts common.NewDBOpts) (driver.AuditTransactionDB, error) {
	return common.NewAuditTransactionDB(db, opts, common.NewTokenInterpreter(postgres.NewInterpreter()))
}

func NewTransactionDB(db *sql.DB, opts common.NewDBOpts) (driver.TokenTransactionDB, error) {
	return common.NewTransactionDB(db, opts, common.NewTokenInterpreter(postgres.NewInterpreter()))
}
