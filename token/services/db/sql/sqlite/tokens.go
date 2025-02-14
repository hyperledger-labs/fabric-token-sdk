/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sqlite

import (
	"database/sql"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/notifier"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/sqlite"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
)

func NewTokenDB(readDB, writeDB *sql.DB, opts common.NewDBOpts) (driver.TokenDB, error) {
	return common.NewTokenDB(readDB, writeDB, opts, common.NewTokenInterpreter(sqlite.NewInterpreter()))
}

func NewTokenNotifier(*sql.DB, *sql.DB, common.NewDBOpts) (driver.TokenNotifier, error) {
	return notifier.NewNotifier(), nil
}
