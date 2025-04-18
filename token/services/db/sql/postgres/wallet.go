/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package postgres

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/postgres"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
)

type WalletDB = common.WalletDB

func NewWalletDB(opts postgres.Opts) (*WalletDB, error) {
	dbs, err := postgres.DbProvider.OpenDB(opts)
	if err != nil {
		return nil, err
	}
	tableNames, err := common.GetTableNames(opts.TablePrefix, opts.TableNameParams...)
	if err != nil {
		return nil, err
	}
	return common.NewWalletDB(dbs.ReadDB, dbs.WriteDB, tableNames)
}
