/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sqlite

import (
	"database/sql"
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/sqlite"
	common2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/common"
)

func mockTransactionsStore(db *sql.DB) *common2.TransactionStore {
	store, _ := common2.NewOwnerTransactionStore(db, db, common2.TableNames{
		Movements:             "MOVEMENTS",
		Transactions:          "TRANSACTIONS",
		Requests:              "REQUESTS",
		Validations:           "VALIDATIONS",
		TransactionEndorseAck: "TRANSACTION_ENDORSE_ACK",
	}, sqlite.NewConditionInterpreter(), sqlite.NewPaginationInterpreter())

	return store
}

var queryConstructorTraits = common2.QueryConstructorTraits{
	SupportsIN:          true,
	MultipleParenthesis: false,
}

func TestGetTokenRequest(t *testing.T) {
	common2.TestGetTokenRequest(t, mockTransactionsStore)
}

func TestQueryMovements(t *testing.T) {
	common2.TestQueryMovements(t, mockTransactionsStore, queryConstructorTraits)
}

func TestQueryTransactions(t *testing.T) {
	common2.TestQueryTransactions(t, mockTransactionsStore)
}

func TestGetStatus(t *testing.T) {
	common2.TestGetStatus(t, mockTransactionsStore)
}

func TestQueryValidations(t *testing.T) {
	common2.TestQueryValidations(t, mockTransactionsStore, queryConstructorTraits)
}

func TestQueryTokenRequests(t *testing.T) {
	common2.TestQueryTokenRequests(t, mockTransactionsStore, queryConstructorTraits)
}

func TestGetTransactionEndorsementAcks(t *testing.T) {
	common2.TestGetTransactionEndorsementAcks(t, mockTransactionsStore)
}

func TestAddTransactionEndorsementAck(t *testing.T) {
	common2.TestAddTransactionEndorsementAck(t, mockTransactionsStore)
}

func TestSetStatus(t *testing.T) {
	common2.TestSetStatus(t, mockTransactionsStore)
}

func TestAWAddTransaction(t *testing.T) {
	common2.TestAWAddTransaction(t, mockTransactionsStore)
}

func TestAWAddTokenRequest(t *testing.T) {
	common2.TestAWAddTokenRequest(t, mockTransactionsStore)
}

func TestAWAddMovement(t *testing.T) {
	common2.TestAWAddMovement(t, mockTransactionsStore)
}

func TestAWAddValidationRecord(t *testing.T) {
	common2.TestAWAddValidationRecord(t, mockTransactionsStore)
}
