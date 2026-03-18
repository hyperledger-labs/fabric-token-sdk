/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package postgres

import (
	common2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/postgres"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/query/pagination"
	common3 "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/common"
)

// AuditTransactionStore is an alias for common.TransactionStore.
type AuditTransactionStore = common3.TransactionStore

// TransactionStore is an alias for common.TransactionStore.
type TransactionStore = common3.TransactionStore

// NewAuditTransactionStore returns a new AuditTransactionStore for the given RWDB and table names.
func NewAuditTransactionStore(dbs *common2.RWDB, tableNames common3.TableNames) (*AuditTransactionStore, error) {
	return common3.NewAuditTransactionStore(dbs.ReadDB, dbs.WriteDB, tableNames, postgres.NewConditionInterpreter(), pagination.NewDefaultInterpreter())
}

// NewTransactionStore returns a new TransactionStore for the given RWDB and table names.
func NewTransactionStore(dbs *common2.RWDB, tableNames common3.TableNames) (*TransactionStore, error) {
	return common3.NewOwnerTransactionStore(dbs.ReadDB, dbs.WriteDB, tableNames, postgres.NewConditionInterpreter(), pagination.NewDefaultInterpreter())
}
