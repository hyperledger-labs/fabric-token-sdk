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

func OpenAuditTransactionDB(k common.Opts) (driver.AuditTransactionDB, error) {
	db, err := sqlite.OpenDB(k.DataSource, k.MaxOpenConns, k.SkipPragmas)
	if err != nil {
		return nil, err
	}
	return NewAuditTransactionDB(db, common.NewDBOptsFromOpts(k))
}

func NewAuditTransactionDB(db *sql.DB, opts common.NewDBOpts) (driver.AuditTransactionDB, error) {
	return common.NewAuditTransactionDB(db, opts)
}

func OpenTransactionDB(k common.Opts) (driver.TokenTransactionDB, error) {
	db, err := sqlite.OpenDB(k.DataSource, k.MaxOpenConns, k.SkipPragmas)
	if err != nil {
		return nil, err
	}
	return NewTransactionDB(db, common.NewDBOptsFromOpts(k))
}

func NewTransactionDB(db *sql.DB, opts common.NewDBOpts) (driver.TokenTransactionDB, error) {
	return common.NewTransactionDB(db, opts)
}
