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
	AuditTransactionDB = common.TransactionDB
	TransactionDB      = common.TransactionDB
)

func NewAuditTransactionDB(opts postgres.Opts) (*AuditTransactionDB, error) {
	dbs, err := postgres.DbProvider.OpenDB(opts)
	if err != nil {
		return nil, err
	}
	tableNames, err := common.GetTableNames(opts.TablePrefix+"_aud", opts.TableNameParams...)
	if err != nil {
		return nil, err
	}
	return common.NewAuditTransactionStore(dbs.ReadDB, dbs.WriteDB, tableNames, common.NewTokenInterpreter(postgres.NewInterpreter()))
}

func NewTransactionDB(opts postgres.Opts) (*TransactionDB, error) {
	dbs, err := postgres.DbProvider.OpenDB(opts)
	if err != nil {
		return nil, err
	}
	tableNames, err := common.GetTableNames(opts.TablePrefix, opts.TableNameParams...)
	if err != nil {
		return nil, err
	}
	return common.NewTransactionDB(dbs.ReadDB, dbs.WriteDB, tableNames, common.NewTokenInterpreter(postgres.NewInterpreter()))
}
