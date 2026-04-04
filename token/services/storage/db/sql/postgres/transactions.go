/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package postgres

import (
	scommon "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/postgres"
	tokensdriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/driver"
	sqlcommon "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/common"
)

// TransactionStore is an alias for common.TransactionStore.
type TransactionStore = sqlcommon.TransactionStore

// AuditTransactionStore is an alias for common.TransactionStore.
type AuditTransactionStore = sqlcommon.TransactionStore

// TransactionNotifier handles notifications for transaction status changes.
type TransactionNotifier struct {
	*Notifier
}

// NewTransactionNotifier returns a new TransactionNotifier for the given RWDB and table names.
func NewTransactionNotifier(dbs *scommon.RWDB, tableNames sqlcommon.TableNames, dataSource string) (*TransactionNotifier, error) {
	return &TransactionNotifier{
		Notifier: NewNotifier(
			dbs.WriteDB,
			tableNames.Requests,
			dataSource,
			[]tokensdriver.Operation{tokensdriver.Update}, // Only listen to UPDATE operations for status changes
			*NewSimplePrimaryKey("tx_id"),
		)}, nil
}

// Subscribe registers a callback function to be called when a transaction request status is updated.
func (n *TransactionNotifier) Subscribe(callback func(tokensdriver.Operation, tokensdriver.TransactionRecordReference)) error {
	return n.Notifier.Subscribe(func(operation tokensdriver.Operation, m map[tokensdriver.ColumnKey]string) {
		callback(operation, tokensdriver.TransactionRecordReference{
			TxID: m["tx_id"],
		})
	})
}

// NewTransactionStoreWithNotifier creates a new TransactionStore with the provided notifier.
func NewTransactionStoreWithNotifier(dbs *scommon.RWDB, tableNames sqlcommon.TableNames, notifier *TransactionNotifier) (*TransactionStore, error) {
	return sqlcommon.NewTransactionStoreWithNotifier(
		dbs.ReadDB,
		dbs.WriteDB,
		tableNames,
		postgres.NewConditionInterpreter(),
		postgres.NewPaginationInterpreter(),
		notifier,
	)
}

// NewAuditTransactionStore creates a new AuditTransactionStore.
func NewAuditTransactionStore(dbs *scommon.RWDB, tableNames sqlcommon.TableNames) (*AuditTransactionStore, error) {
	return sqlcommon.NewAuditTransactionStore(
		dbs.ReadDB,
		dbs.WriteDB,
		tableNames,
		postgres.NewConditionInterpreter(),
		postgres.NewPaginationInterpreter(),
	)
}

// NewTransactionStore creates a new TransactionStore without notifier support.
func NewTransactionStore(dbs *scommon.RWDB, tableNames sqlcommon.TableNames) (*TransactionStore, error) {
	return sqlcommon.NewOwnerTransactionStore(
		dbs.ReadDB,
		dbs.WriteDB,
		tableNames,
		postgres.NewConditionInterpreter(),
		postgres.NewPaginationInterpreter(),
	)
}
