/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package postgres

import (
	common2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/postgres"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/query/pagination"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
)

type (
	AuditTransactionStore = common.TransactionStore
	TransactionStore      = common.TransactionStore
)

func NewAuditTransactionStore(dbs *common2.RWDB, tableNames common.TableNames) (*AuditTransactionStore, error) {
	return common.NewAuditTransactionStore(dbs.ReadDB, dbs.WriteDB, tableNames, postgres.NewConditionInterpreter(), pagination.NewDefaultInterpreter())
}

func NewTransactionStore(dbs *common2.RWDB, tableNames common.TableNames) (*TransactionStore, error) {
	return common.NewOwnerTransactionStore(dbs.ReadDB, dbs.WriteDB, tableNames, postgres.NewConditionInterpreter(), pagination.NewDefaultInterpreter())
}
