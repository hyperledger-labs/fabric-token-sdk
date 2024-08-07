/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package postgres

import (
	common2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/postgres"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
)

func NewWalletDB(k common2.Opts) (driver.WalletDB, error) {
	db, err := postgres.OpenDB(k.DataSource, k.MaxOpenConns)
	if err != nil {
		return nil, err
	}
	return common.NewWalletDB(db, common.NewDBOptsFromOpts(k))
}
