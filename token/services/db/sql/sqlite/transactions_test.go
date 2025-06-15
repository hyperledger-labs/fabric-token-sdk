/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sqlite

import (
	"database/sql"
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/sqlite"
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

func TestGetTokenRequest(t *testing.T) {
	common.TestGetTokenRequest(t, mockTransactionsStore)
}

func TestQueryMovements(t *testing.T) {
	common.TestQueryMovements(t, mockTransactionsStore)
}

func TestQueryTransactions(t *testing.T) {
	common.TestQueryTransactions(t, mockTransactionsStore)
}

func TestGetStatus(t *testing.T) {
	common.TestGetStatus(t, mockTransactionsStore)
}

func TestQueryValidations(t *testing.T) {
	common.TestQueryValidations(t, mockTransactionsStore)
}

func TestQueryTokenRequests(t *testing.T) {
	common.TestQueryTokenRequests(t, mockTransactionsStore)
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
