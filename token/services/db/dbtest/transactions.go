/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dbtest

import (
	"errors"
	"fmt"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/test-go/testify/assert"
)

// This file exposes functions that db driver implementations can use for integration tests
var Cases = []struct {
	Name string
	Fn   func(*testing.T, driver.TokenTransactionDB)
}{
	{"FailsIfRequestDoesNotExist", TFailsIfRequestDoesNotExist},
	{"Status", TStatus},
	{"StoresTimestamp", TStoresTimestamp},
	{"Movements", TMovements},
	{"Transaction", TTransaction},
	{"TokenRequest", TTokenRequest},
	{"AllowsSameTxID", TAllowsSameTxID},
	{"Rollback", TRollback},
	{"TransactionQueries", TTransactionQueries},
	{"ValidationRecordQueries", TValidationRecordQueries},
	{"TEndorserAcks", TEndorserAcks},
}

func TFailsIfRequestDoesNotExist(t *testing.T, db driver.TokenTransactionDB) {
	tx := driver.TransactionRecord{
		TxID:         "tx1",
		ActionType:   driver.Transfer,
		SenderEID:    "bob",
		RecipientEID: "alice",
		TokenType:    "magic",
		Amount:       big.NewInt(10),
		Timestamp:    time.Now(),
		Status:       driver.Pending,
	}
	mv := driver.MovementRecord{
		TxID:         "tx1",
		EnrollmentID: "alice",
		TokenType:    "magic",
		Amount:       big.NewInt(-10),
		Status:       driver.Pending,
	}
	w, _ := db.BeginAtomicWrite()
	err := w.AddTransaction(&tx)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, driver.ErrTokenRequestDoesNotExist))
	w.Rollback()

	w, _ = db.BeginAtomicWrite()
	err = w.AddValidationRecord("tx1", nil)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, driver.ErrTokenRequestDoesNotExist))
	w.Rollback()

	w, _ = db.BeginAtomicWrite()
	err = w.AddMovement(&mv)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, driver.ErrTokenRequestDoesNotExist))
	w.Rollback()
}

func TStatus(t *testing.T, db driver.TokenTransactionDB) {
	tx := driver.TransactionRecord{
		TxID:         "tx1",
		ActionType:   driver.Transfer,
		SenderEID:    "bob",
		RecipientEID: "alice",
		TokenType:    "magic",
		Amount:       big.NewInt(10),
		Timestamp:    time.Now(),
		Status:       driver.Pending,
	}
	mv := driver.MovementRecord{
		TxID:         "tx1",
		EnrollmentID: "alice",
		TokenType:    "magic",
		Amount:       big.NewInt(-10),
		Status:       driver.Pending,
	}

	w, err := db.BeginAtomicWrite()
	assert.NoError(t, err, "begin")
	assert.NoError(t, w.AddTokenRequest("tx1", []byte("request")), "add token request")
	assert.NoError(t, w.AddTransaction(&tx))
	assert.NoError(t, w.AddValidationRecord("tx1", nil), "add validation record")
	assert.NoError(t, w.AddMovement(&mv))
	assert.NoError(t, w.Commit())

	s, mess, err := db.GetStatus("tx1")
	assert.NoError(t, err, "get status error")
	assert.Equal(t, driver.Pending, s, "status should be pending after first creation")
	assert.Equal(t, "", mess)

	txn := getTransactions(t, db, driver.QueryTransactionsParams{})[0]
	assert.Equal(t, driver.Pending, txn.Status, "transaction status should be pending")
	val := getValidationRecords(t, db, driver.QueryValidationRecordsParams{})[0]
	assert.Equal(t, driver.Pending, val.Status, "validation status should be pending")
	mvs, err := db.QueryMovements(driver.QueryMovementsParams{})
	assert.NoError(t, err, "error getting movements")
	assert.Len(t, mvs, 1)
	assert.Equal(t, driver.Pending, mvs[0].Status, "movement status should be pending")

	assert.NoError(t, db.SetStatus("tx1", driver.Confirmed, "message"))
	s, mess, err = db.GetStatus("tx1")
	assert.NoError(t, err)
	assert.Equal(t, driver.Confirmed, s, "status should be changed to confirmed")
	assert.Equal(t, "message", mess)

	txn = getTransactions(t, db, driver.QueryTransactionsParams{})[0]
	assert.Equal(t, driver.Confirmed, txn.Status, "transaction status should be confirmed")
	val = getValidationRecords(t, db, driver.QueryValidationRecordsParams{})[0]
	assert.Equal(t, driver.Confirmed, val.Status, "validation status should be confirmed")
	mvs, err = db.QueryMovements(driver.QueryMovementsParams{})
	assert.NoError(t, err, "error getting movements")
	assert.Len(t, mvs, 1)
	assert.Equal(t, driver.Confirmed, mvs[0].Status, "movement status should be confirmed")
}

func TStoresTimestamp(t *testing.T, db driver.TokenTransactionDB) {
	w, err := db.BeginAtomicWrite()
	assert.NoError(t, err)
	assert.NoError(t, w.AddTokenRequest("tx1", []byte("")))
	assert.NoError(t, w.AddTransaction(&driver.TransactionRecord{
		TxID:         "tx1",
		ActionType:   driver.Transfer,
		SenderEID:    "bob",
		RecipientEID: "alice",
		TokenType:    "magic",
		Amount:       big.NewInt(10),
		Timestamp:    time.Now(),
		Status:       driver.Pending,
	}))
	assert.NoError(t, w.AddValidationRecord("tx1", nil))
	assert.NoError(t, w.Commit())

	now := time.Now()

	// Transaction (timestamp provided)
	txs := getTransactions(t, db, driver.QueryTransactionsParams{})
	assert.Len(t, txs, 1)
	assert.WithinDuration(t, now, txs[0].Timestamp, 3*time.Second)

	// Validation record (timestamp generated by code)
	vr := getValidationRecords(t, db, driver.QueryValidationRecordsParams{})
	assert.Len(t, vr, 1)
	assert.WithinDuration(t, now, vr[0].Timestamp, 3*time.Second)
}

func TMovements(t *testing.T, db driver.TokenTransactionDB) {
	w, err := db.BeginAtomicWrite()
	assert.NoError(t, err)
	assert.NoError(t, w.AddTokenRequest("0", []byte{}))
	assert.NoError(t, w.AddTokenRequest("1", []byte{}))
	assert.NoError(t, w.AddTokenRequest("2", []byte{}))
	assert.NoError(t, w.AddMovement(&driver.MovementRecord{
		TxID:         "0",
		EnrollmentID: "alice",
		TokenType:    "magic",
		Amount:       big.NewInt(10),
	}))
	assert.NoError(t, w.AddMovement(&driver.MovementRecord{
		TxID:         "1",
		EnrollmentID: "alice",
		TokenType:    "magic",
		Amount:       big.NewInt(20),
	}))
	assert.NoError(t, w.AddMovement(&driver.MovementRecord{
		TxID:         "2",
		EnrollmentID: "alice",
		TokenType:    "magic",
		Amount:       big.NewInt(-30),
	}))
	assert.NoError(t, w.Commit())

	// All pending
	records, err := db.QueryMovements(driver.QueryMovementsParams{
		MovementDirection: driver.All,
		SearchDirection:   driver.FromLast,
		TxStatuses:        []driver.TxStatus{driver.Pending},
	})
	assert.NoError(t, err)
	assert.Len(t, records, 3)

	// Received
	records, err = db.QueryMovements(driver.QueryMovementsParams{
		TxStatuses:        []driver.TxStatus{driver.Pending},
		MovementDirection: driver.Received,
		NumRecords:        2,
	})
	assert.NoError(t, err)
	assert.Len(t, records, 2)

	// NumRecords
	records, err = db.QueryMovements(driver.QueryMovementsParams{
		TxStatuses: []driver.TxStatus{driver.Pending},
		NumRecords: 1,
	})
	assert.NoError(t, err)
	assert.Len(t, records, 1)

	assert.NoError(t, db.SetStatus("2", driver.Confirmed, "message"))
	records, err = db.QueryMovements(driver.QueryMovementsParams{TxStatuses: []driver.TxStatus{driver.Pending}, SearchDirection: driver.FromLast, MovementDirection: driver.Received, NumRecords: 3})
	assert.NoError(t, err)
	assert.Len(t, records, 2)

	// setting same status twice should not change the results
	assert.NoError(t, db.SetStatus("2", driver.Confirmed, ""))

	records, err = db.QueryMovements(driver.QueryMovementsParams{TxStatuses: []driver.TxStatus{driver.Confirmed}})
	assert.NoError(t, err)
	assert.Len(t, records, 1)
}

func TTransaction(t *testing.T, db driver.TokenTransactionDB) {
	var txs []*driver.TransactionRecord

	t0 := time.Now()
	lastYear := t0.AddDate(-1, 0, 0)

	w, err := db.BeginAtomicWrite()
	assert.NoError(t, err)
	tr1 := &driver.TransactionRecord{
		TxID:         fmt.Sprintf("tx%d", 99),
		ActionType:   driver.Transfer,
		SenderEID:    "bob",
		RecipientEID: "alice",
		TokenType:    "magic",
		Amount:       big.NewInt(10),
		Timestamp:    lastYear,
	}
	assert.NoError(t, w.AddTokenRequest(tr1.TxID, []byte(fmt.Sprintf("token request for %s", tr1.TxID))))
	assert.NoError(t, w.AddTransaction(tr1))

	for i := 0; i < 20; i++ {
		now := time.Now()
		tr1 := &driver.TransactionRecord{
			TxID:         fmt.Sprintf("tx%d", i),
			ActionType:   driver.Issue,
			SenderEID:    "",
			RecipientEID: "alice",
			TokenType:    "magic",
			Amount:       big.NewInt(10),
			Timestamp:    now,
		}
		assert.NoError(t, w.AddTokenRequest(tr1.TxID, []byte(fmt.Sprintf("token request for %s", tr1.TxID))))
		assert.NoError(t, w.AddTransaction(tr1))
		txs = append(txs, tr1)
	}
	assert.NoError(t, w.Commit())

	// get all except last year's
	t1 := time.Now().Add(time.Second * 3)
	it, err := db.QueryTransactions(driver.QueryTransactionsParams{From: &t0, To: &t1})
	assert.NoError(t, err)
	for _, exp := range txs {
		act, err := it.Next()
		assert.NoError(t, err)
		assertTxEqual(t, exp, act)
	}
	it.Close()

	// get all tx from before the first
	yesterday := t0.AddDate(0, 0, -1).Local().UTC().Truncate(time.Second)
	it, err = db.QueryTransactions(driver.QueryTransactionsParams{To: &yesterday})
	assert.NoError(t, err)
	defer it.Close()

	// find 1 transaction from last year
	tr, err := it.Next()
	assert.NoError(t, err)
	assertTxEqual(t, tr1, tr)

	// find no other transactions
	tr, err = it.Next()
	assert.NoError(t, err)
	assert.Empty(t, tr)

	// update status
	assert.NoError(t, db.SetStatus("tx2", driver.Confirmed, "pineapple"))
	assert.NoError(t, db.SetStatus("tx3", driver.Confirmed, ""))

	status, message, err := db.GetStatus("tx2")
	assert.NoError(t, err)
	assert.Equal(t, driver.Confirmed, status)
	assert.Equal(t, "pineapple", message)

	records := getTransactions(t, db, driver.QueryTransactionsParams{Statuses: []driver.TxStatus{driver.Pending}})
	assert.Len(t, records, 19, "expect 19 pending")

	records = getTransactions(t, db, driver.QueryTransactionsParams{Statuses: []driver.TxStatus{driver.Confirmed}})
	assert.Len(t, records, 2, "expect 2 confirmed")

	status, _, err = db.GetStatus("nonexistenttx")
	assert.NoError(t, err, "a non existent transaction should return Unknown status but no error")
	assert.Equal(t, driver.Unknown, status)
}

const explanation = "transactions [%s]=[%s]"

func assertTxEqual(t *testing.T, exp *driver.TransactionRecord, act *driver.TransactionRecord) {
	if act == nil {
		t.Errorf("expected tx %q, got nil", exp.TxID)
		return
	}
	expl := fmt.Sprintf(explanation, exp.TxID, act.TxID)
	if exp.TxID != act.TxID {
		t.Errorf("expected tx %q, got %q", exp.TxID, act.TxID)
		return
	}

	assert.Equal(t, exp.TxID, act.TxID, expl)
	assert.Equal(t, exp.ActionType, act.ActionType, expl)
	assert.Equal(t, exp.SenderEID, act.SenderEID, expl)
	assert.Equal(t, exp.RecipientEID, act.RecipientEID, expl)
	assert.Equal(t, exp.TokenType, act.TokenType, expl)
	assert.Equal(t, exp.Amount, act.Amount, expl)
	assert.WithinDuration(t, exp.Timestamp, act.Timestamp, 3*time.Second)
}

func TTokenRequest(t *testing.T, db driver.TokenTransactionDB) {
	w, err := db.BeginAtomicWrite()
	assert.NoError(t, err)
	tr := []byte("arbitrary bytes")
	err = w.AddTokenRequest("id1", tr)
	assert.NoError(t, err)
	assert.NoError(t, w.Commit())

	trq, err := db.GetTokenRequest("id1")
	assert.NoError(t, err)
	assert.Equal(t, tr, trq)
}

func TAllowsSameTxID(t *testing.T, db driver.TokenTransactionDB) {
	// bob sends 10 to alice
	tr1 := &driver.TransactionRecord{
		TxID:         "1",
		ActionType:   driver.Transfer,
		SenderEID:    "bob",
		RecipientEID: "alice",
		TokenType:    "magic",
		Amount:       big.NewInt(10),
		Timestamp:    time.Now(),
	}
	// 1 is sent back to bobs wallet as change
	tr2 := &driver.TransactionRecord{
		TxID:         "1",
		ActionType:   driver.Transfer,
		SenderEID:    "bob",
		RecipientEID: "bob",
		TokenType:    "magic",
		Amount:       big.NewInt(1),
		Timestamp:    time.Now(),
	}
	w, err := db.BeginAtomicWrite()
	assert.NoError(t, err)
	assert.NoError(t, w.AddTokenRequest(tr1.TxID, []byte{}))
	assert.NoError(t, w.AddTransaction(tr1))
	assert.NoError(t, w.AddTransaction(tr2))
	assert.NoError(t, w.Commit())

	txs := getTransactions(t, db, driver.QueryTransactionsParams{})
	assert.Len(t, txs, 2)
	assertTxEqual(t, tr1, txs[0])
	assertTxEqual(t, tr2, txs[1])
}

func TRollback(t *testing.T, db driver.TokenTransactionDB) {
	w, err := db.BeginAtomicWrite()
	assert.NoError(t, err)
	assert.NoError(t, w.AddTokenRequest("1", []byte("arbitrary bytes")))

	mr1 := &driver.MovementRecord{
		TxID:         "1",
		EnrollmentID: "bob",
		TokenType:    "magic",
		Amount:       big.NewInt(10),
		Status:       driver.Pending,
	}
	tr1 := &driver.TransactionRecord{
		TxID:         "1",
		ActionType:   driver.Transfer,
		SenderEID:    "bob",
		RecipientEID: "alice",
		TokenType:    "magic",
		Amount:       big.NewInt(10),
		Timestamp:    time.Now().Local().UTC(),
		Status:       driver.Pending,
	}
	assert.NoError(t, w.AddTransaction(tr1))
	assert.NoError(t, w.AddMovement(mr1))
	w.Rollback()
	assert.Len(t, getTransactions(t, db, driver.QueryTransactionsParams{}), 0)
	mvm, err := db.QueryMovements(driver.QueryMovementsParams{})
	assert.NoError(t, err)
	assert.Len(t, mvm, 0)
}

func TTransactionQueries(t *testing.T, db driver.TokenTransactionDB) {
	now := time.Now()
	justBefore := now.Add(-time.Millisecond)
	justAfter := now.Add(time.Millisecond)
	lastYear := now.AddDate(-1, 0, 0)

	tr := []driver.TransactionRecord{
		{
			TxID:         "1",
			Index:        0,
			ActionType:   driver.Issue,
			SenderEID:    "",
			RecipientEID: "bob",
			TokenType:    "magic",
			Amount:       big.NewInt(10),
			Timestamp:    now,
			Status:       driver.Confirmed,
		},
		{
			TxID:         "2",
			Index:        0,
			ActionType:   driver.Transfer,
			SenderEID:    "bob",
			RecipientEID: "alice",
			TokenType:    "magic",
			Amount:       big.NewInt(10),
			Timestamp:    justBefore.Add(-time.Millisecond),
			Status:       driver.Confirmed,
		},
		{
			TxID:         "2",
			Index:        0,
			ActionType:   driver.Transfer,
			SenderEID:    "bob",
			RecipientEID: "bob",
			TokenType:    "magic",
			Amount:       big.NewInt(1),
			Timestamp:    now,
			Status:       driver.Confirmed,
		},
		{
			TxID:         "3",
			Index:        0,
			ActionType:   driver.Transfer,
			SenderEID:    "bob",
			RecipientEID: "alice",
			TokenType:    "magic",
			Amount:       big.NewInt(1),
			Timestamp:    now,
			Status:       driver.Pending,
		},
		{
			TxID:         "4",
			Index:        0,
			ActionType:   driver.Transfer,
			SenderEID:    "bob",
			RecipientEID: "alice",
			TokenType:    "magic",
			Amount:       big.NewInt(1),
			Timestamp:    now,
			Status:       driver.Confirmed,
		},
		{
			TxID:         "5",
			Index:        0,
			ActionType:   driver.Transfer,
			SenderEID:    "bob",
			RecipientEID: "alice",
			TokenType:    "magic",
			Amount:       big.NewInt(1),
			Timestamp:    now,
			Status:       driver.Deleted,
		},
		{
			TxID:         "6",
			Index:        0,
			ActionType:   driver.Transfer,
			SenderEID:    "alice",
			RecipientEID: "bob",
			TokenType:    "abc",
			Amount:       big.NewInt(1),
			Timestamp:    now,
			Status:       driver.Confirmed,
		},
		{
			TxID:         "7",
			Index:        0,
			ActionType:   driver.Transfer,
			SenderEID:    "alice",
			RecipientEID: "bob",
			TokenType:    "abc",
			Amount:       big.NewInt(1),
			Timestamp:    justBefore,
			Status:       driver.Confirmed,
		},
		{
			TxID:         "7",
			Index:        0,
			ActionType:   driver.Transfer,
			SenderEID:    "alice",
			RecipientEID: "dan",
			TokenType:    "abc",
			Amount:       big.NewInt(1),
			Timestamp:    now.AddDate(0, 0, -1),
			Status:       driver.Confirmed,
		},
		{
			TxID:         "8",
			Index:        0,
			ActionType:   driver.Redeem,
			SenderEID:    "dan",
			RecipientEID: "carlos",
			TokenType:    "abc",
			Amount:       big.NewInt(1),
			Timestamp:    now.AddDate(0, 0, -1),
			Status:       driver.Confirmed,
		},
		{
			TxID:         "9",
			Index:        0,
			ActionType:   driver.Transfer,
			SenderEID:    "alice",
			RecipientEID: "dan",
			TokenType:    "abc",
			Amount:       big.NewInt(1),
			Timestamp:    now.AddDate(0, 0, 1),
			Status:       driver.Confirmed,
		},
		{
			TxID:         "10",
			Index:        0,
			ActionType:   driver.Redeem,
			SenderEID:    "alice",
			RecipientEID: "",
			TokenType:    "abc",
			Amount:       big.NewInt(1),
			Timestamp:    now.AddDate(0, 0, 1),
			Status:       driver.Confirmed,
		},
	}
	testCases := []struct {
		name        string
		params      driver.QueryTransactionsParams
		expectedLen int
		expectedSql string
	}{
		{
			name:        "No params",
			params:      driver.QueryTransactionsParams{},
			expectedLen: len(tr),
		},
		{
			name: "Only driver.Confirmed",
			params: driver.QueryTransactionsParams{
				Statuses: []driver.TxStatus{driver.Confirmed},
			},
			expectedLen: 10,
		},
		{
			name: "Pending and deleted",
			params: driver.QueryTransactionsParams{
				Statuses: []driver.TxStatus{driver.Pending, driver.Deleted},
			},
			expectedLen: 2,
		},
		{
			name: "Confirmed from alice should return all driver.Confirmed",
			params: driver.QueryTransactionsParams{
				SenderWallet: "alice",
				Statuses:     []driver.TxStatus{driver.Confirmed},
			},
			expectedLen: 10,
		},
		{
			name: "Recipient matches should return all",
			params: driver.QueryTransactionsParams{
				RecipientWallet: "alice",
			},
			expectedLen: 12,
		},
		{
			name: "Sender OR recipient matches",
			params: driver.QueryTransactionsParams{
				SenderWallet:    "alice",
				RecipientWallet: "alice",
			},
			expectedLen: 9,
		},
		{
			name: "Sender OR recipient matches, from last year",
			params: driver.QueryTransactionsParams{
				SenderWallet:    "alice",
				RecipientWallet: "alice",
				From:            &lastYear,
			},
			expectedLen: 9,
		},
		{
			name: "Only this millisecond",
			params: driver.QueryTransactionsParams{
				From: &justBefore,
				To:   &justAfter,
			},
			expectedLen: 7,
		},
		{
			name: "Only this millisecond for alice",
			params: driver.QueryTransactionsParams{
				SenderWallet:    "alice",
				RecipientWallet: "alice",
				From:            &justBefore,
				To:              &justAfter,
			},
			expectedLen: 5,
		},
		{
			name: "Get redemption",
			params: driver.QueryTransactionsParams{
				ActionTypes: []driver.ActionType{driver.Redeem},
			},
			expectedLen: 2,
		},
	}

	w, err := db.BeginAtomicWrite()
	assert.NoError(t, err)
	var previous string
	for _, r := range tr {
		if r.TxID != previous {
			assert.NoError(t, w.AddTokenRequest(r.TxID, []byte{}))
		}
		assert.NoError(t, w.AddTransaction(&r))
		previous = r.TxID
	}
	assert.NoError(t, w.Commit())
	for _, r := range tr {
		if r.Status != driver.Pending {
			assert.NoError(t, db.SetStatus(r.TxID, r.Status, ""))
		}
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := getTransactions(t, db, tc.params)
			assert.Len(t, res, tc.expectedLen, fmt.Sprintf("params: %v", tc.params))
		})
	}
}

func getTransactions(t *testing.T, db driver.TokenTransactionDB, params driver.QueryTransactionsParams) []*driver.TransactionRecord {
	records, err := db.QueryTransactions(params)
	assert.NoError(t, err)
	defer records.Close()
	var txs []*driver.TransactionRecord
	for {
		r, err := records.Next()
		assert.NoError(t, err)
		if r == nil {
			return txs
		}
		txs = append(txs, r)
	}
}

func TValidationRecordQueries(t *testing.T, db driver.TokenTransactionDB) {
	beforeTx := time.Now().UTC().Add(-1 * time.Second)
	exp := []driver.ValidationRecord{
		{
			TxID:         "1",
			TokenRequest: []byte("tr1"),
			Metadata: map[string][]byte{
				"key": []byte("value"),
			},
			Status: driver.Unknown,
		},
		{
			TxID:         "2",
			TokenRequest: []byte{},
			Metadata:     nil,
			Status:       driver.Unknown,
		},
		{
			TxID:         "3",
			TokenRequest: []byte("tr3"),
			Metadata: map[string][]byte{
				"key": []byte("value"),
			},
			Status: driver.Unknown,
		},
		{
			TxID:         "4",
			TokenRequest: []byte("tr4"),
			Metadata: map[string][]byte{
				"key": []byte("value"),
			},
			Status: driver.Confirmed,
		},
	}
	w, err := db.BeginAtomicWrite()
	assert.NoError(t, err)
	for _, e := range exp {
		assert.NoError(t, w.AddTokenRequest(e.TxID, e.TokenRequest))
		assert.NoError(t, w.AddValidationRecord(e.TxID, e.Metadata), "AddValidationRecord "+e.TxID)
	}
	assert.NoError(t, w.Commit(), "Commit")
	for _, e := range exp {
		if e.Status != driver.Pending {
			assert.NoError(t, db.SetStatus(e.TxID, e.Status, ""))
		}
	}
	all := getValidationRecords(t, db, driver.QueryValidationRecordsParams{})
	assert.Len(t, all, 4)

	for i, vr := range exp {
		assert.Equal(t, vr.TxID, all[i].TxID, fmt.Sprintf("%v", all[i]))
		assert.Equal(t, vr.TokenRequest, all[i].TokenRequest, fmt.Sprintf("%v - %d", all[i], len(all[i].TokenRequest)))
		assert.Equal(t, vr.Metadata, all[i].Metadata, fmt.Sprintf("%v", all[i]))
		assert.Equal(t, vr.Status, all[i].Status, fmt.Sprintf("%v", all[i]))
		assert.WithinDuration(t, beforeTx, all[i].Timestamp, 5*time.Second, fmt.Sprintf("%v", all[i]))
	}

	to := getValidationRecords(t, db, driver.QueryValidationRecordsParams{
		To: &beforeTx,
	})
	assert.Len(t, to, 0, "Expect no results if all records are created after 'To'")

	from := getValidationRecords(t, db, driver.QueryValidationRecordsParams{
		From: &beforeTx,
	})
	assert.Len(t, from, len(exp), "'From' before creation should include all records'")

	confirmed := getValidationRecords(t, db, driver.QueryValidationRecordsParams{
		Statuses: []driver.TxStatus{driver.Confirmed},
	})
	assert.Len(t, confirmed, 1)

	filtered := getValidationRecords(t, db, driver.QueryValidationRecordsParams{
		Filter: func(r *driver.ValidationRecord) bool {
			return r.Status == driver.Unknown
		},
	})
	assert.Len(t, filtered, 3)
}

func getValidationRecords(t *testing.T, db driver.TokenTransactionDB, params driver.QueryValidationRecordsParams) []*driver.ValidationRecord {
	records, err := db.QueryValidations(params)
	assert.NoError(t, err)
	defer records.Close()
	var txs []*driver.ValidationRecord
	for {
		r, err := records.Next()
		assert.NoError(t, err)
		if r == nil {
			return txs
		}
		txs = append(txs, r)
	}
}

func TEndorserAcks(t *testing.T, db driver.TokenTransactionDB) {
	createTransaction(t, db, "1")
	wg := sync.WaitGroup{}
	n := 100
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			assert.NoError(t, db.AddTransactionEndorsementAck("1", []byte(fmt.Sprintf("alice_%d", i)), []byte(fmt.Sprintf("sigma_%d", i))))
			acks, err := db.GetTransactionEndorsementAcks("1")
			assert.NoError(t, err)
			assert.True(t, len(acks) != 0)
			wg.Done()
		}(i)
	}
	wg.Wait()

	acks, err := db.GetTransactionEndorsementAcks("1")
	assert.NoError(t, err)
	assert.Len(t, acks, n)
	for i := 0; i < n; i++ {
		assert.Equal(t, []byte(fmt.Sprintf("sigma_%d", i)), acks[view.Identity(fmt.Sprintf("alice_%d", i)).String()])
	}
}

func createTransaction(t *testing.T, db driver.TokenTransactionDB, txID string) {
	w, err := db.BeginAtomicWrite()
	if err != nil {
		t.Fatalf("error creating transaction while trying to test something else: %w", err)
	}
	if err := w.AddTokenRequest(txID, []byte{}); err != nil {
		t.Fatalf("error creating token request while trying to test something else: %w", err)
	}
	tr1 := &driver.TransactionRecord{
		TxID:         txID,
		ActionType:   driver.Transfer,
		SenderEID:    "bob",
		RecipientEID: "alice",
		TokenType:    "magic",
		Amount:       big.NewInt(10),
		Timestamp:    time.Now().Local().UTC(),
		Status:       driver.Pending,
	}
	if err := w.AddTransaction(tr1); err != nil {
		t.Fatalf("error creating transaction while trying to test something else: %w", err)
	}
	if err := w.Commit(); err != nil {
		t.Fatalf("error committing transaction while trying to test something else: %w", err)
	}
}
