/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package postgres

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/postgres"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
)

type (
	TokenDB       = common.TokenDB
	TokenNotifier = postgres.Notifier
)

func NewTokenDB(opts postgres.Opts) (*TokenDB, error) {
	readWriteDB, err := postgres.OpenDB(opts)
	if err != nil {
		return nil, err
	}
	tableNames, err := common.GetTableNames(opts.TablePrefix)
	if err != nil {
		return nil, err
	}
	return common.NewTokenDB(readWriteDB, readWriteDB, tableNames, common.NewTokenInterpreter(postgres.NewInterpreter()))
}

func NewTokenNotifier(opts postgres.Opts) (*TokenNotifier, error) {
	readWriteDB, err := postgres.OpenDB(opts)
	if err != nil {
		return nil, err
	}
	tables, err := common.GetTableNames(opts.TablePrefix)
	if err != nil {
		return nil, err
	}
	return postgres.NewNotifier(readWriteDB, tables.Tokens, opts.DataSource, postgres.AllOperations, *postgres.NewSimplePrimaryKey("tx_id"), *postgres.NewSimplePrimaryKey("idx")), nil

}
