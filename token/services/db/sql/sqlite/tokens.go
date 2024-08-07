/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sqlite

import (
	common2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/sqlite"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
)

func NewTokenDB(k common2.Opts) (driver.TokenDB, error) {
	db, err := sqlite.OpenDB(k.DataSource, k.MaxOpenConns, k.SkipPragmas)
	if err != nil {
		return nil, err
	}
	return common.NewTokenDB(db, common.NewDBOptsFromOpts(k))
}

func NewTokenNDB(k common2.Opts) (driver.TokenNDB, error) {
	panic("unimplemented")
}
