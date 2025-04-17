/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sqlite

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/sqlite"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
)

type IdentityDB = common.IdentityDB

func NewIdentityDB(opts sqlite.Opts) (*IdentityDB, error) {
	readDB, writeDB, err := sqlite.OpenRWDBs(opts)
	if err != nil {
		return nil, err
	}
	tableNames, err := common.GetTableNames(opts.TablePrefix)
	if err != nil {
		return nil, err
	}
	return common.NewCachedIdentityDB(readDB, writeDB, tableNames, sqlite.NewInterpreter())
}
