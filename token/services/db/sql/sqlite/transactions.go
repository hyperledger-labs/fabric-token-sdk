/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sqlite

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/sqlite"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	common2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
)

func NewAuditTransactionDB(k common.Opts) (driver.AuditTransactionDB, error) {
	db, err := sqlite.OpenDB(k.DataSource, k.MaxOpenConns, k.SkipPragmas)
	if err != nil {
		return nil, err
	}
	return common2.NewAuditTransactionDB(db, common2.NewDBOptsFromOpts(k))
}

func NewTransactionDB(k common.Opts) (driver.TokenTransactionDB, error) {
	db, err := sqlite.OpenDB(k.DataSource, k.MaxOpenConns, k.SkipPragmas)
	if err != nil {
		return nil, err
	}
	return common2.NewTransactionDB(db, common2.NewDBOptsFromOpts(k))
}
