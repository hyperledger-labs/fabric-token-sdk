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

func NewIdentityDB(readDB, writeDB *sql.DB, opts common.NewDBOpts) (driver.IdentityDB, error) {
	return common.NewCachedIdentityDB(readDB, writeDB, opts, postgres.NewInterpreter())
}
