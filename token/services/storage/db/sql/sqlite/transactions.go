/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sqlite

import (
	common2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/query/pagination"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/sqlite"
	common3 "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/common"
)

type (
	AuditTransactionStore = common3.TransactionStore
	OwnerTransactionStore = common3.TransactionStore
)

func NewAuditTransactionStore(dbs *common2.RWDB, tableNames common3.TableNames) (*AuditTransactionStore, error) {
	return common3.NewAuditTransactionStore(dbs.ReadDB, dbs.WriteDB, tableNames, sqlite.NewConditionInterpreter(), pagination.NewDefaultInterpreter())
}

func NewTransactionStore(dbs *common2.RWDB, tableNames common3.TableNames) (*OwnerTransactionStore, error) {
	return common3.NewOwnerTransactionStore(dbs.ReadDB, dbs.WriteDB, tableNames, sqlite.NewConditionInterpreter(), pagination.NewDefaultInterpreter())
}
