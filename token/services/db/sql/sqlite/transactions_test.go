/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sqlite

import (
	"database/sql"
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/sqlite"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
)

func mockTransactionsStore(db *sql.DB) *common.TransactionStore {
	store, _ := common.NewOwnerTransactionStore(db, db, common.TableNames{
		Movements:             "MOVEMENTS",
		Transactions:          "TRANSACTIONS",
		Requests:              "REQUESTS",
		Validations:           "VALIDATIONS",
		TransactionEndorseAck: "TRANSACTION_ENDORSE_ACK",
	}, sqlite.NewConditionInterpreter(), sqlite.NewPaginationInterpreter())
	return store
}

var queryConstructorTraits = common.QueryConstructorTraits{
	SupportsIN:          true,
	MultipleParenthesis: false,
}

func TestGetTokenRequest(t *testing.T) {
	common.TestGetTokenRequest(t, mockTransactionsStore)
}

func TestQueryMovements(t *testing.T) {
	common.TestQueryMovements(t, mockTransactionsStore, queryConstructorTraits)
}

func TestQueryTransactions(t *testing.T) {
	common.TestQueryTransactions(t, mockTransactionsStore)
}

func TestGetStatus(t *testing.T) {
	common.TestGetStatus(t, mockTransactionsStore)
}

func TestQueryValidations(t *testing.T) {
	common.TestQueryValidations(t, mockTransactionsStore, queryConstructorTraits)
}

func TestQueryTokenRequests(t *testing.T) {
	common.TestQueryTokenRequests(t, mockTransactionsStore, queryConstructorTraits)
}

func TestGetTransactionEndorsementAcks(t *testing.T) {
	common.TestGetTransactionEndorsementAcks(t, mockTransactionsStore)
}

func TestAddTransactionEndorsementAck(t *testing.T) {
	common.TestAddTransactionEndorsementAck(t, mockTransactionsStore)
}

func TestSetStatus(t *testing.T) {
	common.TestSetStatus(t, mockTransactionsStore)
}

func TestAWAddTransaction(t *testing.T) {
	common.TestAWAddTransaction(t, mockTransactionsStore)
}

func TestAWAddTokenRequest(t *testing.T) {
	common.TestAWAddTokenRequest(t, mockTransactionsStore)
}

func TestAWAddMovement(t *testing.T) {
	common.TestAWAddMovement(t, mockTransactionsStore)
}

func TestAWAddValidationRecord(t *testing.T) {
	common.TestAWAddValidationRecord(t, mockTransactionsStore)
}
