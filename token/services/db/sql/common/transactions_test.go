/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common_test

import (
	"context"
	"database/sql"
	driver2 "database/sql/driver"
	"math/big"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/query/pagination"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/sqlite"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	. "github.com/onsi/gomega"
)

func TestGetTokenRequest(t *testing.T) {
	RegisterTestingT(t)
	db, mockDB, err := sqlmock.New()
	Expect(err).ToNot(HaveOccurred())

	input := string("an_id")
	output := []byte("some_result")
	mockDB.
		ExpectQuery("SELECT request FROM REQUESTS WHERE tx_id = \\$1").
		WithArgs(input).
		WillReturnRows(mockDB.NewRows([]string{"tx_id"}).AddRow(output))

	info, err := mockTransactionsStore(db).GetTokenRequest(context.Background(), input)

	Expect(mockDB.ExpectationsWereMet()).To(Succeed())
	Expect(err).ToNot(HaveOccurred())
	Expect(info).To(Equal(output))
}

func TestQueryMovements(t *testing.T) {
	RegisterTestingT(t)
	db, mockDB, err := sqlmock.New()
	Expect(err).ToNot(HaveOccurred())

	record := driver.MovementRecord{
		TxID:         "1234",
		EnrollmentID: "5678",
		TokenType:    token.Type("USD"),
		Amount:       big.NewInt(-100),
		Status:       driver.TxStatus(3),
	}
	output := []driver2.Value{
		record.TxID, record.EnrollmentID, record.TokenType, int(record.Amount.Int64()), record.Status,
	}
	mockDB.
		ExpectQuery("SELECT MOVEMENTS.tx_id, enrollment_id, token_type, amount, REQUESTS.status "+
			"FROM MOVEMENTS LEFT JOIN REQUESTS ON MOVEMENTS.tx_id = REQUESTS.tx_id "+
			"WHERE \\(enrollment_id = \\$1\\) AND \\(token_type = \\$2\\) AND \\(status = \\$3\\) AND \\(amount < \\$4\\) "+
			"ORDER BY stored_at DESC "+
			"LIMIT \\$5").
		WithArgs(record.EnrollmentID, record.TokenType, record.Status, 0, 1).
		WillReturnRows(mockDB.NewRows([]string{"tx_id", "enrollment_id", "token_type", "amount", "status"}).AddRow(output...))

	info, err := mockTransactionsStore(db).QueryMovements(context.Background(),
		driver.QueryMovementsParams{
			EnrollmentIDs:     []string{record.EnrollmentID},
			TokenTypes:        []token.Type{record.TokenType},
			TxStatuses:        []driver.TxStatus{record.Status},
			MovementDirection: driver.Sent,
			NumRecords:        1})

	Expect(mockDB.ExpectationsWereMet()).To(Succeed())
	Expect(err).ToNot(HaveOccurred())
	Expect(*info[0]).To(Equal(record))
}

func TestQueryTransactions(t *testing.T) {
	RegisterTestingT(t)
	db, mockDB, err := sqlmock.New()
	Expect(err).ToNot(HaveOccurred())

	record := driver.TransactionRecord{
		TxID:         "1234",
		ActionType:   driver.Transfer,
		SenderEID:    "alice",
		RecipientEID: "bob",
		TokenType:    token.Type("USD"),
		Amount:       big.NewInt(100),
		Status:       driver.TxStatus(3),
	}
	output := []driver2.Value{
		record.TxID, record.ActionType, record.SenderEID, record.RecipientEID, record.TokenType, int(record.Amount.Int64()), record.Status, 0, 0, time.Time{},
	}
	mockDB.
		ExpectQuery("SELECT TRANSACTIONS.tx_id, action_type, sender_eid, recipient_eid, token_type, amount, " +
			"REQUESTS.status, REQUESTS.application_metadata, REQUESTS.public_metadata, stored_at " +
			"FROM TRANSACTIONS LEFT JOIN REQUESTS ON TRANSACTIONS.tx_id = REQUESTS.tx_id ORDER BY stored_at ASC").
		WillReturnRows(mockDB.NewRows([]string{"tx_id", "action_type", "sender_eid", "recipient_eid", "token_type", "amount", "status", "application_metadata", "public_metadata", "stored_at"}).AddRow(output...))

	info, err := mockTransactionsStore(db).QueryTransactions(context.Background(),
		driver.QueryTransactionsParams{
			IDs: []string{}}, pagination.None())

	Expect(mockDB.ExpectationsWereMet()).To(Succeed())
	Expect(err).ToNot(HaveOccurred())
	actualRecord, err := info.Items.Next()
	Expect(*actualRecord).To(Equal(record))
}

func TestGetStatus(t *testing.T) {
	RegisterTestingT(t)
	db, mockDB, err := sqlmock.New()
	Expect(err).ToNot(HaveOccurred())

	input := string("1234")
	output := []driver2.Value{3, string("some_message")}

	mockDB.
		ExpectQuery("SELECT status, status_message FROM REQUESTS WHERE tx_id = \\$1").
		WithArgs(input).
		WillReturnRows(mockDB.NewRows([]string{"status", "status_message"}).AddRow(output...))

	status, statusMessage, err := mockTransactionsStore(db).GetStatus(context.Background(), input)

	Expect(mockDB.ExpectationsWereMet()).To(Succeed())
	Expect(err).ToNot(HaveOccurred())
	Expect(status).To(Equal(output[0]))
	Expect(statusMessage).To(Equal(output[1]))
}

func TestQueryValidations(t *testing.T) {
	RegisterTestingT(t)
	db, mockDB, err := sqlmock.New()
	Expect(err).ToNot(HaveOccurred())

	time1 := time.Date(2025, time.June, 8, 10, 0, 0, 0, time.UTC)
	time2 := time.Date(2025, time.June, 9, 10, 0, 0, 0, time.UTC)
	record := driver.ValidationRecord{
		TxID:         "1234",
		TokenRequest: []byte("some request"),
		Timestamp:    time1,
		Status:       driver.TxStatus(3),
	}
	output := []driver2.Value{
		record.TxID, record.TokenRequest, 0, record.Status, record.Timestamp,
	}
	mockDB.
		ExpectQuery("SELECT VALIDATIONS.tx_id, REQUESTS.request, metadata, REQUESTS.status, VALIDATIONS.stored_at "+
			"FROM VALIDATIONS LEFT JOIN REQUESTS ON VALIDATIONS.tx_id = REQUESTS.tx_id "+
			"WHERE \\(\\(stored_at >= \\$1\\) AND \\(stored_at <= \\$2\\)\\) AND \\(\\(status\\) IN \\(\\(\\$3\\), \\(\\$4\\)\\)\\)").
		WithArgs(time1, time2, 3, 4).
		WillReturnRows(mockDB.NewRows([]string{"tx_id", "request", "metadata", "status", "stored_at"}).AddRow(output...))

	records, err := mockTransactionsStore(db).QueryValidations(context.Background(),
		driver.QueryValidationRecordsParams{
			From:     &time1,
			To:       &time2,
			Statuses: []driver.TxStatus{3, 4},
		},
	)

	Expect(mockDB.ExpectationsWereMet()).To(Succeed())
	Expect(err).ToNot(HaveOccurred())
	actualRecord, err := records.Next()
	Expect(*actualRecord).To(Equal(record))
}

func TestQueryTokenRequests(t *testing.T) {
	RegisterTestingT(t)
	db, mockDB, err := sqlmock.New()
	Expect(err).ToNot(HaveOccurred())

	record := driver.TokenRequestRecord{
		TxID:         "1234",
		TokenRequest: []byte("some request"),
		Status:       driver.TxStatus(3),
	}
	output := []driver2.Value{
		record.TxID, record.TokenRequest, record.Status,
	}
	mockDB.
		ExpectQuery("SELECT tx_id, request, status FROM REQUESTS WHERE \\(status\\) IN \\(\\(\\$1\\), \\(\\$2\\)\\)").
		WithArgs(3, 4).
		WillReturnRows(mockDB.NewRows([]string{"tx_id", "request", "status"}).AddRow(output...))

	records, err := mockTransactionsStore(db).QueryTokenRequests(context.Background(),
		driver.QueryTokenRequestsParams{
			Statuses: []driver.TxStatus{3, 4},
		},
	)

	Expect(mockDB.ExpectationsWereMet()).To(Succeed())
	Expect(err).ToNot(HaveOccurred())
	actualRecord, err := records.Next()
	Expect(*actualRecord).To(Equal(record))
}

func mockTransactionsStore(db *sql.DB) *common.TransactionStore {
	return common.NewTransactionStore(db, db, common.TransactionTables{
		Movements:             "MOVEMENTS",
		Transactions:          "TRANSACTIONS",
		Requests:              "REQUESTS",
		Validations:           "VALIDATIONS",
		TransactionEndorseAck: "TRANSACTION_ENDORSE_ACK",
	}, sqlite.NewConditionInterpreter(), sqlite.NewPaginationInterpreter())
}
