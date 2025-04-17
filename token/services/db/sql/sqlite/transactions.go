/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sqlite

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/sqlite"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
)

type (
	AuditTransactionDB = common.TransactionDB
	TransactionDB      = common.TransactionDB
)

func NewAuditTransactionDB(opts sqlite.Opts) (*AuditTransactionDB, error) {
	readDB, writeDB, err := sqlite.OpenRWDBs(opts)
	if err != nil {
		return nil, err
	}
	tableNames, err := common.GetTableNames(opts.TablePrefix+"_aud", opts.TableNameParams...)
	if err != nil {
		return nil, err
	}
	return common.NewAuditTransactionDB(readDB, writeDB, tableNames, common.NewTokenInterpreter(sqlite.NewInterpreter()))
}

func NewTransactionDB(opts sqlite.Opts) (*TransactionDB, error) {
	readDB, writeDB, err := sqlite.OpenRWDBs(opts)
	if err != nil {
		return nil, err
	}
	tableNames, err := common.GetTableNames(opts.TablePrefix, opts.TableNameParams...)
	if err != nil {
		return nil, err
	}
	return common.NewTransactionDB(readDB, writeDB, tableNames, common.NewTokenInterpreter(sqlite.NewInterpreter()))
}
