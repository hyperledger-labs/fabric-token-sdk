/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"context"
	"database/sql"
	driver2 "database/sql/driver"
	"math/big"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections/iterators"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/query/pagination"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/onsi/gomega"
)

type transactionsStoreConstructor func(*sql.DB) *TransactionStore

func TestGetTokenRequest(t *testing.T, store transactionsStoreConstructor) {
	gomega.RegisterTestingT(t)
	db, mockDB, err := sqlmock.New()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	input := string("1234")
	output := []byte("some_result")
	mockDB.
		ExpectQuery("SELECT request FROM REQUESTS WHERE tx_id = \\$1").
		WithArgs(input).
		WillReturnRows(mockDB.NewRows([]string{"request"}).AddRow(output))

	info, err := store(db).GetTokenRequest(context.Background(), input)

	gomega.Expect(mockDB.ExpectationsWereMet()).To(gomega.Succeed())
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	gomega.Expect(info).To(gomega.Equal(output))
}

func TestQueryMovements(t *testing.T, store transactionsStoreConstructor) {
	gomega.RegisterTestingT(t)
	db, mockDB, err := sqlmock.New()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	record := driver.MovementRecord{
		TxID:         "1234",
		EnrollmentID: "5678",
		TokenType:    token.Type("USD"),
		Amount:       big.NewInt(-100),
		Status:       driver.Deleted,
	}
	output := []driver2.Value{
		record.TxID, record.EnrollmentID, record.TokenType, int(record.Amount.Int64()), record.Status,
	}
	mockDB.
		ExpectQuery("SELECT MOVEMENTS.tx_id, enrollment_id, token_type, amount, REQUESTS.status "+
			"FROM MOVEMENTS LEFT JOIN REQUESTS ON MOVEMENTS.tx_id = REQUESTS.tx_id "+
			"WHERE \\(\\(\\(enrollment_id = \\$1\\)\\)\\) AND \\(\\(\\(token_type = \\$2\\)\\)\\) AND \\(\\(\\(status = \\$3\\)\\)\\) AND \\(amount < \\$4\\) "+
			"ORDER BY stored_at DESC "+
			"LIMIT \\$5").
		WithArgs(record.EnrollmentID, record.TokenType, record.Status, 0, 1).
		WillReturnRows(mockDB.NewRows([]string{"tx_id", "enrollment_id", "token_type", "amount", "status"}).AddRow(output...))

	info, err := store(db).QueryMovements(context.Background(),
		driver.QueryMovementsParams{
			EnrollmentIDs:     []string{record.EnrollmentID},
			TokenTypes:        []token.Type{record.TokenType},
			TxStatuses:        []driver.TxStatus{record.Status},
			MovementDirection: driver.Sent,
			NumRecords:        1,
		})

	gomega.Expect(mockDB.ExpectationsWereMet()).To(gomega.Succeed())
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	gomega.Expect(info).To(gomega.ConsistOf(&record))
}

func TestQueryTransactions(t *testing.T, store transactionsStoreConstructor) {
	gomega.RegisterTestingT(t)
	db, mockDB, err := sqlmock.New()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	record := driver.TransactionRecord{
		TxID:         "1234",
		ActionType:   driver.Transfer,
		SenderEID:    "alice",
		RecipientEID: "bob",
		TokenType:    token.Type("USD"),
		Amount:       big.NewInt(100),
		Status:       driver.Deleted,
	}
	output := []driver2.Value{
		record.TxID, record.ActionType, record.SenderEID, record.RecipientEID, record.TokenType, int(record.Amount.Int64()), record.Status, nil, nil, time.Time{},
	}
	mockDB.
		ExpectQuery("SELECT TRANSACTIONS.tx_id, action_type, sender_eid, recipient_eid, token_type, amount, " +
			"REQUESTS.status, REQUESTS.application_metadata, REQUESTS.public_metadata, stored_at " +
			"FROM TRANSACTIONS LEFT JOIN REQUESTS ON TRANSACTIONS.tx_id = REQUESTS.tx_id ORDER BY stored_at ASC").
		WillReturnRows(mockDB.NewRows([]string{"tx_id", "action_type", "sender_eid", "recipient_eid", "token_type", "amount", "status", "application_metadata", "public_metadata", "stored_at"}).AddRow(output...))

	info, err := store(db).QueryTransactions(context.Background(),
		driver.QueryTransactionsParams{
			IDs: []string{}}, pagination.None())

	gomega.Expect(mockDB.ExpectationsWereMet()).To(gomega.Succeed())
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	records, err := iterators.ReadAllValues(info.Items)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	gomega.Expect(records).To(gomega.ConsistOf(record))
}

func TestGetStatus(t *testing.T, store transactionsStoreConstructor) {
	gomega.RegisterTestingT(t)
	db, mockDB, err := sqlmock.New()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	input := "1234"
	output := []driver2.Value{3, "some_message"}

	mockDB.
		ExpectQuery("SELECT status, status_message FROM REQUESTS WHERE tx_id = \\$1").
		WithArgs(input).
		WillReturnRows(mockDB.NewRows([]string{"status", "status_message"}).AddRow(output...))

	status, statusMessage, err := store(db).GetStatus(context.Background(), input)

	gomega.Expect(mockDB.ExpectationsWereMet()).To(gomega.Succeed())
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	gomega.Expect(status).To(gomega.Equal(output[0]))
	gomega.Expect(statusMessage).To(gomega.Equal(output[1]))
}

func TestQueryValidations(t *testing.T, store transactionsStoreConstructor) {
	gomega.RegisterTestingT(t)
	db, mockDB, err := sqlmock.New()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	timeFrom := time.Date(2025, time.June, 8, 10, 0, 0, 0, time.UTC)
	timeTo := time.Date(2025, time.June, 9, 10, 0, 0, 0, time.UTC)
	record := driver.ValidationRecord{
		TxID:         "1234",
		TokenRequest: []byte("some request"),
		Timestamp:    timeFrom,
		Status:       driver.Deleted,
	}
	output := []driver2.Value{
		record.TxID, record.TokenRequest, nil, record.Status, record.Timestamp,
	}
	mockDB.
		ExpectQuery("SELECT VALIDATIONS.tx_id, REQUESTS.request, metadata, REQUESTS.status, VALIDATIONS.stored_at "+
			"FROM VALIDATIONS LEFT JOIN REQUESTS ON VALIDATIONS.tx_id = REQUESTS.tx_id "+
			"WHERE \\(\\(stored_at >= \\$1\\) AND \\(stored_at <= \\$2\\)\\) AND \\(\\(\\(status = \\$3\\)\\) OR \\(\\(status = \\$4\\)\\)\\)").
		WithArgs(timeFrom, timeTo, driver.Deleted, driver.Unknown).
		WillReturnRows(mockDB.NewRows([]string{"tx_id", "request", "metadata", "status", "stored_at"}).AddRow(output...))

	it, err := store(db).QueryValidations(context.Background(),
		driver.QueryValidationRecordsParams{
			From:     &timeFrom,
			To:       &timeTo,
			Statuses: []driver.TxStatus{driver.Deleted, driver.Unknown},
		},
	)

	gomega.Expect(mockDB.ExpectationsWereMet()).To(gomega.Succeed())
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	records, err := iterators.ReadAllValues(it)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	gomega.Expect(records).To(gomega.ConsistOf(record))
}

func TestQueryTokenRequests(t *testing.T, store transactionsStoreConstructor) {
	gomega.RegisterTestingT(t)
	db, mockDB, err := sqlmock.New()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	record := driver.TokenRequestRecord{
		TxID:         "1234",
		TokenRequest: []byte("some request"),
		Status:       driver.Deleted,
	}
	output := []driver2.Value{
		record.TxID, record.TokenRequest, record.Status,
	}
	mockDB.
		ExpectQuery("SELECT tx_id, request, status FROM REQUESTS WHERE \\(\\(status = \\$1\\)\\) OR \\(\\(status = \\$2\\)\\)").
		WithArgs(driver.Deleted, driver.Unknown).
		WillReturnRows(mockDB.NewRows([]string{"tx_id", "request", "status"}).AddRow(output...))

	it, err := store(db).QueryTokenRequests(context.Background(),
		driver.QueryTokenRequestsParams{
			Statuses: []driver.TxStatus{driver.Deleted, driver.Unknown},
		},
	)

	gomega.Expect(mockDB.ExpectationsWereMet()).To(gomega.Succeed())
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	records, err := iterators.ReadAllValues[driver.TokenRequestRecord](it)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	gomega.Expect(records).To(gomega.ConsistOf(record))
}

func TestGetTransactionEndorsementAcks(t *testing.T, store transactionsStoreConstructor) {
	gomega.RegisterTestingT(t)
	db, mockDB, err := sqlmock.New()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	record := struct {
		endorser string
		sigma    []byte
	}{
		endorser: "auditor",
		sigma:    []byte("5678"),
	}
	inputID := "1234"
	output := []driver2.Value{record.endorser, record.sigma}

	mockDB.
		ExpectQuery("SELECT endorser, sigma FROM TRANSACTION_ENDORSE_ACK WHERE tx_id = \\$1").
		WithArgs(inputID).
		WillReturnRows(mockDB.NewRows([]string{"endorser", "sigma"}).AddRow(output...))

	acks, err := store(db).GetTransactionEndorsementAcks(context.Background(), inputID)

	gomega.Expect(mockDB.ExpectationsWereMet()).To(gomega.Succeed())
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	gomega.Expect(acks).To(gomega.HaveLen(1))
	gomega.Expect(acks).To(gomega.HaveKeyWithValue(token2.Identity(record.endorser).UniqueID(), record.sigma))
}

func TestAddTransactionEndorsementAck(t *testing.T, store transactionsStoreConstructor) {
	gomega.RegisterTestingT(t)
	db, mockDB, err := sqlmock.New()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	uuid := sqlmock.AnyArg()
	txID := "1234"
	eID := token2.Identity([]byte("5678"))
	sigma := []byte("signature")
	now := sqlmock.AnyArg()

	mockDB.ExpectExec("INSERT INTO TRANSACTION_ENDORSE_ACK \\(id, tx_id, endorser, sigma, stored_at\\) VALUES \\(\\$1, \\$2, \\$3, \\$4, \\$5\\)").
		WithArgs(uuid, txID, eID, sigma, now).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = store(db).AddTransactionEndorsementAck(context.Background(), txID, eID, sigma)

	gomega.Expect(mockDB.ExpectationsWereMet()).To(gomega.Succeed())
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
}

func TestSetStatus(t *testing.T, store transactionsStoreConstructor) {
	gomega.RegisterTestingT(t)
	db, mockDB, err := sqlmock.New()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	txID := "1234"
	status := driver.Confirmed
	message := "message"

	mockDB.ExpectExec("UPDATE REQUESTS SET status = \\$1, status_message = \\$2 WHERE tx_id = \\$3").
		WithArgs(status, message, txID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = store(db).SetStatus(context.Background(), txID, status, message)

	gomega.Expect(mockDB.ExpectationsWereMet()).To(gomega.Succeed())
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
}

type AnyUUID struct{}

func (a AnyUUID) Match(v driver2.Value) bool {
	_, ok := v.(string)
	return ok
}

func TestAWAddTransaction(t *testing.T, store transactionsStoreConstructor) {
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

	aw, err := store(db).BeginAtomicWrite()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	gomega.Expect(aw.AddTransaction(context.Background(), input)).To(gomega.Succeed())
	gomega.Expect(aw.Commit()).To(gomega.Succeed())

	gomega.Expect(mockDB.ExpectationsWereMet()).To(gomega.Succeed())
}

func TestAWAddTokenRequest(t *testing.T, store transactionsStoreConstructor) {
	gomega.RegisterTestingT(t)
	db, mockDB, err := sqlmock.New()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	txID := "txid"
	tr := []byte("1234")
	ppHash := []byte("5678")
	status := driver.Pending
	status_message := ""

	mockDB.ExpectBegin()
	mockDB.
		ExpectExec("INSERT INTO REQUESTS \\(tx_id, request, status, status_message, application_metadata, public_metadata, pp_hash\\) "+
			"VALUES \\(\\$1, \\$2, \\$3, \\$4, \\$5, \\$6, \\$7\\)").
		WithArgs(txID, tr, status, status_message, "{}", "{}", ppHash).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mockDB.ExpectCommit()

	aw, err := store(db).BeginAtomicWrite()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	gomega.Expect(aw.AddTokenRequest(context.Background(), txID, tr, nil, nil, ppHash)).To(gomega.Succeed())
	gomega.Expect(aw.Commit()).To(gomega.Succeed())

	gomega.Expect(mockDB.ExpectationsWereMet()).To(gomega.Succeed())
}

func TestAWAddMovement(t *testing.T, store transactionsStoreConstructor) {
	gomega.RegisterTestingT(t)
	db, mockDB, err := sqlmock.New()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	input := &driver.MovementRecord{
		TxID:         "txid",
		EnrollmentID: "EID",
		TokenType:    "USD",
		Amount:       big.NewInt(10),
		Status:       driver.Pending,
	}
	now := sqlmock.AnyArg()

	mockDB.ExpectBegin()
	mockDB.
		ExpectExec("INSERT INTO MOVEMENTS \\(id, tx_id, enrollment_id, token_type, amount, stored_at\\) "+
			"VALUES \\(\\$1, \\$2, \\$3, \\$4, \\$5, \\$6\\)").
		WithArgs(AnyUUID{}, input.TxID, input.EnrollmentID, input.TokenType, input.Amount.Int64(), now).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mockDB.ExpectCommit()

	aw, err := store(db).BeginAtomicWrite()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	gomega.Expect(aw.AddMovement(context.Background(), input)).To(gomega.Succeed())
	gomega.Expect(aw.Commit()).To(gomega.Succeed())

	gomega.Expect(mockDB.ExpectationsWereMet()).To(gomega.Succeed())
}

func TestAWAddValidationRecord(t *testing.T, store transactionsStoreConstructor) {
	gomega.RegisterTestingT(t)
	db, mockDB, err := sqlmock.New()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	txID := "txid"
	now := sqlmock.AnyArg()

	mockDB.ExpectBegin()
	mockDB.
		ExpectExec("INSERT INTO VALIDATIONS \\(tx_id, metadata, stored_at\\) VALUES \\(\\$1, \\$2, \\$3\\)").
		WithArgs(txID, "null", now).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mockDB.ExpectCommit()

	aw, err := store(db).BeginAtomicWrite()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	gomega.Expect(aw.AddValidationRecord(context.Background(), txID, nil)).To(gomega.Succeed())
	gomega.Expect(aw.Commit()).To(gomega.Succeed())

	gomega.Expect(mockDB.ExpectationsWereMet()).To(gomega.Succeed())
}
