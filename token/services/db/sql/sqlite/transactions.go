/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sqlite

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/query/pagination"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/sqlite"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
)

type (
	AuditTransactionStore = common.TransactionStore
	OwnerTransactionStore = common.TransactionStore
)

func NewAuditTransactionStore(opts sqlite.Opts) (*AuditTransactionStore, error) {
	dbs, err := sqlite.DbProvider.OpenDB(opts)
	if err != nil {
		return nil, err
	}
	tableNames, err := common.GetTableNames(opts.TablePrefix+"_aud", opts.TableNameParams...)
	if err != nil {
		return nil, err
	}
	return common.NewAuditTransactionStore(dbs.ReadDB, dbs.WriteDB, tableNames, sqlite.NewConditionInterpreter(), pagination.NewDefaultInterpreter())
}

func NewTransactionStore(opts sqlite.Opts) (*OwnerTransactionStore, error) {
	dbs, err := sqlite.DbProvider.OpenDB(opts)
	if err != nil {
		return nil, err
	}
	tableNames, err := common.GetTableNames(opts.TablePrefix, opts.TableNameParams...)
	if err != nil {
		return nil, err
	}
	return common.NewOwnerTransactionStore(dbs.ReadDB, dbs.WriteDB, tableNames, sqlite.NewConditionInterpreter(), pagination.NewDefaultInterpreter())
}
