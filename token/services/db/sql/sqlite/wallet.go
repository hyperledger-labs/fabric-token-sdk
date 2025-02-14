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

func NewWalletDB(readDB, writeDB *sql.DB, opts common.NewDBOpts) (driver.WalletDB, error) {
	return common.NewWalletDB(readDB, writeDB, opts)
}
