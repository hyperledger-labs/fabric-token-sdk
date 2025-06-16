/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package postgres

import (
	"context"
	"database/sql"
	driver2 "database/sql/driver"
	"math/big"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/sqlite"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
	"github.com/onsi/gomega"
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

type AnyUUID struct{}

func (a AnyUUID) Match(v driver2.Value) bool {
	_, ok := v.(string)
	return ok
}

func TestAddTransaction(t *testing.T) {
	gomega.RegisterTestingT(t)
	db, mockDB, err := sqlmock.New()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	input := &driver.TransactionRecord{
		TxID:         "txid",
		ActionType:   driver.Transfer,
		SenderEID:    "sender",
		RecipientEID: "recipient",
		TokenType:    "USD",
		Amount:       big.NewInt(10),
		Timestamp:    time.Now(),
	}

	mockDB.ExpectBegin()
	mockDB.
		ExpectExec("INSERT INTO TRANSACTIONS \\(id, tx_id, action_type, sender_eid, recipient_eid, token_type, amount, stored_at\\) VALUES \\(\\$1, \\$2, \\$3, \\$4, \\$5, \\$6, \\$7, \\$8\\)").
		WithArgs(AnyUUID{}, input.TxID, 1, input.SenderEID, input.RecipientEID, input.TokenType, input.Amount.Int64(), input.Timestamp.UTC()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mockDB.ExpectCommit()

	aw, err := mockTransactionsStore(db).BeginAtomicWrite()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	gomega.Expect(aw.AddTransaction(context.Background(), input)).To(gomega.Succeed())
	gomega.Expect(aw.Commit()).To(gomega.Succeed())

	gomega.Expect(mockDB.ExpectationsWereMet()).To(gomega.Succeed())
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
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
