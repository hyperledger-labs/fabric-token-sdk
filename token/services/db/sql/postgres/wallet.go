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

func OpenWalletDB(k common.Opts) (driver.WalletDB, error) {
	db, err := postgres.OpenDB(k.DataSource, k.MaxOpenConns, k.MaxIdleConns, k.MaxIdleTime)
	if err != nil {
		return nil, err
	}
	return NewWalletDB(db, common.NewDBOptsFromOpts(k))
}

func NewWalletDB(db *sql.DB, opts common.NewDBOpts) (driver.WalletDB, error) {
	return common.NewWalletDB(db, opts)
}
