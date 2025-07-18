/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dbtest

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections/iterators"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/query/pagination"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	driver3 "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/utils"
	"github.com/test-go/testify/assert"
)

func TransactionsTest(t *testing.T, cfgProvider cfgProvider) {
	for _, c := range tokenTransactionDBCases {
		driver := cfgProvider(c.Name)
		db, err := driver.NewOwnerTransaction("", c.Name)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(c.Name, func(xt *testing.T) {
			defer utils.IgnoreError(db.Close)
			c.Fn(xt, db)
		})
	}
}

var tokenTransactionDBCases = []struct {
	Name string
	Fn   func(*testing.T, driver3.TokenTransactionStore)
}{
	// {"FailsIfRequestDoesNotExist", TFailsIfRequestDoesNotExist},
	// {"Status", TStatus},
	// {"StoresTimestamp", TStoresTimestamp},
	// {"Movements", TMovements},
	// {"Transaction", TTransaction},
	// {"TokenRequest", TTokenRequest},
	// {"AllowsSameTxID", TAllowsSameTxID},
	// {"Rollback", TRollback},
	{"TransactionQueries", TTransactionQueries},
	// {"ValidationRecordQueries", TValidationRecordQueries},
	// {"TEndorserAcks", TEndorserAcks},
}

func TFailsIfRequestDoesNotExist(t *testing.T, db driver3.TokenTransactionStore) {
	ctx := context.Background()
	tx := driver3.TransactionRecord{
		TxID:         "tx1",
		ActionType:   driver3.Transfer,
		SenderEID:    "bob",
		RecipientEID: "alice",
		TokenType:    "magic",
		Amount:       big.NewInt(10),
		Timestamp:    time.Now(),
		Status:       driver3.Pending,
	}
	mv := driver3.MovementRecord{
		TxID:         "tx1",
		EnrollmentID: "alice",
		TokenType:    "magic",
		Amount:       big.NewInt(-10),
		Status:       driver3.Pending,
	}
	w, _ := db.BeginAtomicWrite()
	err := w.AddTransaction(ctx, tx)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, driver3.ErrTokenRequestDoesNotExist))
	w.Rollback()

	w, _ = db.BeginAtomicWrite()
	err = w.AddValidationRecord(ctx, "tx1", nil)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, driver3.ErrTokenRequestDoesNotExist))
	w.Rollback()

	w, _ = db.BeginAtomicWrite()
	err = w.AddMovement(ctx, mv)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, driver3.ErrTokenRequestDoesNotExist))
	w.Rollback()
}

func TStatus(t *testing.T, db driver3.TokenTransactionStore) {
	ctx := context.Background()
	tx := driver3.TransactionRecord{
		TxID:         "tx1",
		ActionType:   driver3.Transfer,
		SenderEID:    "bob",
		RecipientEID: "alice",
		TokenType:    "magic",
		Amount:       big.NewInt(10),
		Timestamp:    time.Now(),
		Status:       driver3.Pending,
	}
	mv := driver3.MovementRecord{
		TxID:         "tx1",
		EnrollmentID: "alice",
		TokenType:    "magic",
		Amount:       big.NewInt(-10),
		Status:       driver3.Pending,
	}

	w, err := db.BeginAtomicWrite()
	assert.NoError(t, err, "begin")
	assert.NoError(t, w.AddTokenRequest(ctx, "tx1", []byte("request"), map[string][]byte{}, nil, driver2.PPHash("tr")), "add token request")
	assert.NoError(t, w.AddTransaction(ctx, tx))
	assert.NoError(t, w.AddValidationRecord(ctx, "tx1", nil), "add validation record")
	assert.NoError(t, w.AddMovement(ctx, mv))
	assert.NoError(t, w.Commit())

	s, mess, err := db.GetStatus(ctx, "tx1")
	assert.NoError(t, err, "get status error")
	assert.Equal(t, driver3.Pending, s, "status should be pending after first creation")
	assert.Equal(t, "", mess)

	txn := getTransactions(t, db, driver3.QueryTransactionsParams{})[0]
	assert.Equal(t, driver3.Pending, txn.Status, "transaction status should be pending")
	val := getValidationRecords(t, db, driver3.QueryValidationRecordsParams{})[0]
	assert.Equal(t, driver3.Pending, val.Status, "validation status should be pending")
	mvs, err := db.QueryMovements(ctx, driver3.QueryMovementsParams{})
	assert.NoError(t, err, "error getting movements")
	assert.Len(t, mvs, 1)
	assert.Equal(t, driver3.Pending, mvs[0].Status, "movement status should be pending")

	assert.NoError(t, db.SetStatus(context.TODO(), "tx1", driver3.Confirmed, "message"))
	s, mess, err = db.GetStatus(ctx, "tx1")
	assert.NoError(t, err)
	assert.Equal(t, driver3.Confirmed, s, "status should be changed to confirmed")
	assert.Equal(t, "message", mess)

	txn = getTransactions(t, db, driver3.QueryTransactionsParams{})[0]
	assert.Equal(t, driver3.Confirmed, txn.Status, "transaction status should be confirmed")
	val = getValidationRecords(t, db, driver3.QueryValidationRecordsParams{})[0]
	assert.Equal(t, driver3.Confirmed, val.Status, "validation status should be confirmed")
	mvs, err = db.QueryMovements(ctx, driver3.QueryMovementsParams{})
	assert.NoError(t, err, "error getting movements")
	assert.Len(t, mvs, 1)
	assert.Equal(t, driver3.Confirmed, mvs[0].Status, "movement status should be confirmed")
}

func TStoresTimestamp(t *testing.T, db driver3.TokenTransactionStore) {
	ctx := context.Background()
	w, err := db.BeginAtomicWrite()
	assert.NoError(t, err)
	assert.NoError(t, w.AddTokenRequest(ctx, "tx1", []byte(""), map[string][]byte{}, nil, driver2.PPHash("tr")))
	assert.NoError(t, w.AddTransaction(ctx, driver3.TransactionRecord{
		TxID:         "tx1",
		ActionType:   driver3.Transfer,
		SenderEID:    "bob",
		RecipientEID: "alice",
		TokenType:    "magic",
		Amount:       big.NewInt(10),
		Timestamp:    time.Now(),
		Status:       driver3.Pending,
	}))
	assert.NoError(t, w.AddValidationRecord(ctx, "tx1", nil))
	assert.NoError(t, w.Commit())

	now := time.Now()

	// Transaction (timestamp provided)
	txs := getTransactions(t, db, driver3.QueryTransactionsParams{})
	assert.Len(t, txs, 1)
	assert.WithinDuration(t, now, txs[0].Timestamp, 3*time.Second)

	// Validation record (timestamp generated by code)
	vr := getValidationRecords(t, db, driver3.QueryValidationRecordsParams{})
	assert.Len(t, vr, 1)
	assert.WithinDuration(t, now, vr[0].Timestamp, 3*time.Second)
}

func TMovements(t *testing.T, db driver3.TokenTransactionStore) {
	ctx := context.Background()
	w, err := db.BeginAtomicWrite()
	assert.NoError(t, err)
	assert.NoError(t, w.AddTokenRequest(ctx, "0", []byte{}, map[string][]byte{}, nil, driver2.PPHash("tr")))
	assert.NoError(t, w.AddTokenRequest(ctx, "1", []byte{}, map[string][]byte{}, nil, driver2.PPHash("tr")))
	assert.NoError(t, w.AddTokenRequest(ctx, "2", []byte{}, map[string][]byte{}, nil, driver2.PPHash("tr")))
	assert.NoError(t, w.AddMovement(ctx, driver3.MovementRecord{
		TxID:         "0",
		EnrollmentID: "alice",
		TokenType:    "magic",
		Amount:       big.NewInt(10),
	}))
	assert.NoError(t, w.AddMovement(ctx, driver3.MovementRecord{
		TxID:         "1",
		EnrollmentID: "alice",
		TokenType:    "magic",
		Amount:       big.NewInt(20),
	}))
	assert.NoError(t, w.AddMovement(ctx, driver3.MovementRecord{
		TxID:         "2",
		EnrollmentID: "alice",
		TokenType:    "magic",
		Amount:       big.NewInt(-30),
	}))
	assert.NoError(t, w.Commit())

	// All pending
	records, err := db.QueryMovements(ctx, driver3.QueryMovementsParams{
		MovementDirection: driver3.All,
		SearchDirection:   driver3.FromLast,
		TxStatuses:        []driver3.TxStatus{driver3.Pending},
	})
	assert.NoError(t, err)
	assert.Len(t, records, 3)

	// Received
	records, err = db.QueryMovements(ctx, driver3.QueryMovementsParams{
		TxStatuses:        []driver3.TxStatus{driver3.Pending},
		MovementDirection: driver3.Received,
		NumRecords:        2,
	})
	assert.NoError(t, err)
	assert.Len(t, records, 2)

	// NumRecords
	records, err = db.QueryMovements(ctx, driver3.QueryMovementsParams{
		TxStatuses: []driver3.TxStatus{driver3.Pending},
		NumRecords: 1,
	})
	assert.NoError(t, err)
	assert.Len(t, records, 1)

	assert.NoError(t, db.SetStatus(context.TODO(), "2", driver3.Confirmed, "message"))
	records, err = db.QueryMovements(ctx, driver3.QueryMovementsParams{TxStatuses: []driver3.TxStatus{driver3.Pending}, SearchDirection: driver3.FromLast, MovementDirection: driver3.Received, NumRecords: 3})
	assert.NoError(t, err)
	assert.Len(t, records, 2)

	// setting same status twice should not change the results
	assert.NoError(t, db.SetStatus(context.TODO(), "2", driver3.Confirmed, ""))

	records, err = db.QueryMovements(ctx, driver3.QueryMovementsParams{TxStatuses: []driver3.TxStatus{driver3.Confirmed}})
	assert.NoError(t, err)
	assert.Len(t, records, 1)
}

func TTransaction(t *testing.T, db driver3.TokenTransactionStore) {
	ctx := context.Background()
	var txs []driver3.TransactionRecord

	t0 := time.Now()
	lastYear := t0.AddDate(-1, 0, 0)

	w, err := db.BeginAtomicWrite()
	assert.NoError(t, err)
	tr1 := driver3.TransactionRecord{
		TxID:                fmt.Sprintf("tx%d", 99),
		ActionType:          driver3.Transfer,
		SenderEID:           "bob",
		RecipientEID:        "alice",
		TokenType:           "magic",
		Amount:              big.NewInt(10),
		ApplicationMetadata: map[string][]byte{},
		Timestamp:           lastYear,
	}
	assert.NoError(t, w.AddTokenRequest(ctx, tr1.TxID, []byte(fmt.Sprintf("token request for %s", tr1.TxID)), map[string][]byte{}, nil, driver2.PPHash("tr")))
	assert.NoError(t, w.AddTransaction(ctx, tr1))

	pm := map[string][]byte{"key": []byte("val")}

	for i := 0; i < 20; i++ {
		now := time.Now()
		ctx := context.Background()
		tr := driver3.TransactionRecord{
			TxID:         fmt.Sprintf("tx%d", i),
			ActionType:   driver3.Issue,
			SenderEID:    "",
			RecipientEID: "alice",
			TokenType:    "magic",
			Amount:       big.NewInt(10),
			Timestamp:    now,
			ApplicationMetadata: map[string][]byte{
				"this is the first key":  {99, 33, 22, 11},
				"this is the second key": []byte("with some text as the value " + fmt.Sprintf("tx%d", i)),
			},
			PublicMetadata: pm,
		}
		assert.NoError(t, w.AddTokenRequest(ctx, tr.TxID, []byte(fmt.Sprintf("token request for %s", tr.TxID)), tr.ApplicationMetadata, pm, driver2.PPHash("tr")))
		assert.NoError(t, w.AddTransaction(ctx, tr))
		txs = append(txs, tr)
	}
	assert.NoError(t, w.Commit())

	// get one
	one := getTransactions(t, db, driver3.QueryTransactionsParams{IDs: []string{"tx10"}})
	assert.Len(t, one, 1)
	assert.Equal(t, "tx10", one[0].TxID)

	// get all except last year's
	t1 := time.Now().Add(time.Second * 3)
	it, err := db.QueryTransactions(ctx, driver3.QueryTransactionsParams{From: &t0, To: &t1}, pagination.None())
	assert.NoError(t, err)
	for _, exp := range txs {
		act, err := it.Items.Next()
		assert.NoError(t, err)
		assertTxEqual(t, &exp, act)
	}
	it.Items.Close()

	// get all tx from before the first
	yesterday := t0.AddDate(0, 0, -1).Local().UTC().Truncate(time.Second)
	it, err = db.QueryTransactions(ctx, driver3.QueryTransactionsParams{To: &yesterday}, pagination.None())
	assert.NoError(t, err)
	defer it.Items.Close()

	// find 1 transaction from last year
	tr, err := it.Items.Next()
	assert.NoError(t, err)
	assertTxEqual(t, &tr1, tr)

	// find no other transactions
	tr, err = it.Items.Next()
	assert.NoError(t, err)
	assert.Empty(t, tr)

	// update status
	assert.NoError(t, db.SetStatus(context.TODO(), "tx2", driver3.Confirmed, "pineapple"))
	assert.NoError(t, db.SetStatus(context.TODO(), "tx3", driver3.Confirmed, ""))

	status, message, err := db.GetStatus(context.Background(), "tx2")
	assert.NoError(t, err)
	assert.Equal(t, driver3.Confirmed, status)
	assert.Equal(t, "pineapple", message)

	records := getTransactions(t, db, driver3.QueryTransactionsParams{Statuses: []driver3.TxStatus{driver3.Pending}})
	assert.Len(t, records, 19, "expect 19 pending")

	records = getTransactions(t, db, driver3.QueryTransactionsParams{Statuses: []driver3.TxStatus{driver3.Confirmed}})
	assert.Len(t, records, 2, "expect 2 confirmed")

	status, _, err = db.GetStatus(ctx, "nonexistenttx")
	assert.NoError(t, err, "a non existent transaction should return Unknown status but no error")
	assert.Equal(t, driver3.Unknown, status)

	// exclude to self
	w, err = db.BeginAtomicWrite()
	assert.NoError(t, err)
	tr1 = driver3.TransactionRecord{
		TxID:                "1234",
		ActionType:          driver3.Transfer,
		SenderEID:           "alice",
		RecipientEID:        "alice",
		TokenType:           "magic",
		Amount:              big.NewInt(10),
		ApplicationMetadata: map[string][]byte{},
		Timestamp:           lastYear,
	}
	assert.NoError(t, w.AddTokenRequest(ctx, tr1.TxID, fmt.Appendf(nil, "token request for %s", tr1.TxID), map[string][]byte{}, nil, driver2.PPHash("tr")))
	assert.NoError(t, w.AddTransaction(ctx, tr1))
	assert.NoError(t, w.Commit())
	noChange := getTransactions(t, db, driver3.QueryTransactionsParams{ExcludeToSelf: true})
	assert.Len(t, noChange, 21)
	for _, tr := range noChange {
		assert.NotEqual(t, tr.TxID, tr1.TxID, "transaction to self should not be included")
	}
}

const explanation = "transactions [%s]=[%s]"

func assertTxEqual(t *testing.T, exp *driver3.TransactionRecord, act *driver3.TransactionRecord) {
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
	assert.Equal(t, exp.ApplicationMetadata, act.ApplicationMetadata, expl)
	assert.WithinDuration(t, exp.Timestamp, act.Timestamp, 3*time.Second, expl)
	if len(exp.PublicMetadata) > 0 {
		assert.True(t, reflect.DeepEqual(exp.PublicMetadata, act.PublicMetadata), expl)
	}
}

func TTokenRequest(t *testing.T, db driver3.TokenTransactionStore) {
	ctx := context.Background()
	w, err := db.BeginAtomicWrite()
	assert.NoError(t, err)
	tr1 := []byte("arbitrary bytes")
	err = w.AddTokenRequest(ctx, "id1", tr1, map[string][]byte{}, nil, []byte("tr"))
	assert.NoError(t, err)
	tr2 := []byte("arbitrary bytes 2")
	err = w.AddTokenRequest(ctx, "id2", tr2, map[string][]byte{}, nil, []byte("tr"))
	assert.NoError(t, err)
	assert.NoError(t, w.Commit())
	assert.NoError(t, db.SetStatus(context.TODO(), "id2", driver3.Confirmed, ""))

	trq, err := db.GetTokenRequest(ctx, "id1")
	assert.NoError(t, err)
	assert.Equal(t, tr1, trq)
	trq, err = db.GetTokenRequest(ctx, "id2")
	assert.NoError(t, err)
	assert.Equal(t, tr2, trq)

	// iterate over all
	it, err := db.QueryTokenRequests(ctx, driver3.QueryTokenRequestsParams{})
	assert.NoError(t, err)
	counter := 0
	for {
		record, err := it.Next()
		assert.NoError(t, err)
		if record == nil {
			break
		}
		if record.TxID == "id1" {
			assert.Equal(t, tr1, record.TokenRequest)
			assert.Equal(t, driver3.Pending, record.Status)
			counter++
			continue
		}
		if record.TxID == "id2" {
			assert.Equal(t, tr2, record.TokenRequest)
			assert.Equal(t, driver3.Confirmed, record.Status)
			counter++
			continue
		}
	}
	assert.Equal(t, 2, counter)
	it.Close()

	// iterate over pending and confirmed
	it, err = db.QueryTokenRequests(ctx, driver3.QueryTokenRequestsParams{Statuses: []driver3.TxStatus{driver3.Confirmed, driver3.Pending}})
	assert.NoError(t, err)
	counter = 0
	for {
		record, err := it.Next()
		assert.NoError(t, err)
		if record == nil {
			break
		}
		if record.TxID == "id1" {
			assert.Equal(t, tr1, record.TokenRequest)
			assert.Equal(t, driver3.Pending, record.Status)
			counter++
			continue
		}
		if record.TxID == "id2" {
			assert.Equal(t, tr2, record.TokenRequest)
			assert.Equal(t, driver3.Confirmed, record.Status)
			counter++
			continue
		}
	}
	assert.Equal(t, 2, counter)
	it.Close()

	// iterator over confirmed
	it, err = db.QueryTokenRequests(ctx, driver3.QueryTokenRequestsParams{Statuses: []driver3.TxStatus{driver3.Confirmed}})
	assert.NoError(t, err)
	record, err := it.Next()
	assert.NoError(t, err)
	assert.Equal(t, tr2, record.TokenRequest)
	assert.Equal(t, driver3.Confirmed, record.Status)
	record, err = it.Next()
	assert.NoError(t, err)
	assert.Nil(t, record)
	it.Close()

	// iterator over pending
	it, err = db.QueryTokenRequests(ctx, driver3.QueryTokenRequestsParams{Statuses: []driver3.TxStatus{driver3.Pending}})
	assert.NoError(t, err)
	record, err = it.Next()
	assert.NoError(t, err)
	assert.Equal(t, tr1, record.TokenRequest)
	assert.Equal(t, driver3.Pending, record.Status)
	record, err = it.Next()
	assert.NoError(t, err)
	assert.Nil(t, record)
	it.Close()

	// iterator over deleted
	it, err = db.QueryTokenRequests(ctx, driver3.QueryTokenRequestsParams{Statuses: []driver3.TxStatus{driver3.Deleted}})
	assert.NoError(t, err)
	record, err = it.Next()
	assert.NoError(t, err)
	assert.Nil(t, record)
	it.Close()
}

func TAllowsSameTxID(t *testing.T, db driver3.TokenTransactionStore) {
	ctx := context.Background()

	// bob sends 10 to alice
	tr1 := driver3.TransactionRecord{
		TxID:                "1",
		ActionType:          driver3.Transfer,
		SenderEID:           "bob",
		RecipientEID:        "alice",
		TokenType:           "magic",
		ApplicationMetadata: map[string][]byte{},
		Amount:              big.NewInt(10),
		Timestamp:           time.Now(),
	}
	// 1 is sent back to bobs wallet as change
	tr2 := driver3.TransactionRecord{
		TxID:                "1",
		ActionType:          driver3.Transfer,
		SenderEID:           "bob",
		RecipientEID:        "bob",
		TokenType:           "magic",
		ApplicationMetadata: map[string][]byte{},
		Amount:              big.NewInt(1),
		Timestamp:           time.Now(),
	}
	w, err := db.BeginAtomicWrite()
	assert.NoError(t, err)
	assert.NoError(t, w.AddTokenRequest(ctx, tr1.TxID, []byte{}, map[string][]byte{}, nil, driver2.PPHash("tr")))
	assert.NoError(t, w.AddTransaction(ctx, tr1))
	assert.NoError(t, w.AddTransaction(ctx, tr2))
	assert.NoError(t, w.Commit())

	txs := getTransactions(t, db, driver3.QueryTransactionsParams{})
	assert.Len(t, txs, 2)
	assertTxEqual(t, &tr1, txs[0])
	assertTxEqual(t, &tr2, txs[1])
}

func TRollback(t *testing.T, db driver3.TokenTransactionStore) {
	ctx := context.Background()
	w, err := db.BeginAtomicWrite()
	assert.NoError(t, err)
	assert.NoError(t, w.AddTokenRequest(ctx, "1", []byte("arbitrary bytes"), map[string][]byte{}, nil, driver2.PPHash("tr")))

	mr1 := driver3.MovementRecord{
		TxID:         "1",
		EnrollmentID: "bob",
		TokenType:    "magic",
		Amount:       big.NewInt(10),
		Status:       driver3.Pending,
	}
	tr1 := driver3.TransactionRecord{
		TxID:         "1",
		ActionType:   driver3.Transfer,
		SenderEID:    "bob",
		RecipientEID: "alice",
		TokenType:    "magic",
		Amount:       big.NewInt(10),
		Timestamp:    time.Now().Local().UTC(),
		Status:       driver3.Pending,
	}
	assert.NoError(t, w.AddTransaction(ctx, tr1))
	assert.NoError(t, w.AddMovement(ctx, mr1))
	w.Rollback()
	assert.Len(t, getTransactions(t, db, driver3.QueryTransactionsParams{}), 0)
	mvm, err := db.QueryMovements(ctx, driver3.QueryMovementsParams{})
	assert.NoError(t, err)
	assert.Len(t, mvm, 0)
}

func TTransactionQueries(t *testing.T, db driver3.TokenTransactionStore) {
	now := time.Now()
	justBefore := now.Add(-time.Millisecond)
	justAfter := now.Add(time.Millisecond)
	lastYear := now.AddDate(-1, 0, 0)
	ctx := context.Background()

	tr := []driver3.TransactionRecord{
		{
			TxID:         "1",
			ActionType:   driver3.Issue,
			SenderEID:    "",
			RecipientEID: "bob",
			TokenType:    "magic",
			Amount:       big.NewInt(10),
			Timestamp:    now,
			Status:       driver3.Confirmed,
		},
		{
			TxID:         "2",
			ActionType:   driver3.Transfer,
			SenderEID:    "bob",
			RecipientEID: "alice",
			TokenType:    "magic",
			Amount:       big.NewInt(10),
			Timestamp:    justBefore.Add(-time.Millisecond),
			Status:       driver3.Confirmed,
		},
		{
			TxID:         "2",
			ActionType:   driver3.Transfer,
			SenderEID:    "bob",
			RecipientEID: "bob",
			TokenType:    "magic",
			Amount:       big.NewInt(1),
			Timestamp:    now,
			Status:       driver3.Confirmed,
		},
		{
			TxID:         "3",
			ActionType:   driver3.Transfer,
			SenderEID:    "bob",
			RecipientEID: "alice",
			TokenType:    "magic",
			Amount:       big.NewInt(1),
			Timestamp:    now,
			Status:       driver3.Pending,
		},
		{
			TxID:         "4",
			ActionType:   driver3.Transfer,
			SenderEID:    "bob",
			RecipientEID: "alice",
			TokenType:    "magic",
			Amount:       big.NewInt(1),
			Timestamp:    now,
			Status:       driver3.Confirmed,
		},
		{
			TxID:         "5",
			ActionType:   driver3.Transfer,
			SenderEID:    "bob",
			RecipientEID: "alice",
			TokenType:    "magic",
			Amount:       big.NewInt(1),
			Timestamp:    now,
			Status:       driver3.Deleted,
		},
		{
			TxID:         "6",
			ActionType:   driver3.Transfer,
			SenderEID:    "alice",
			RecipientEID: "bob",
			TokenType:    "abc",
			Amount:       big.NewInt(1),
			Timestamp:    now,
			Status:       driver3.Confirmed,
		},
		{
			TxID:         "7",
			ActionType:   driver3.Transfer,
			SenderEID:    "alice",
			RecipientEID: "bob",
			TokenType:    "abc",
			Amount:       big.NewInt(1),
			Timestamp:    justBefore,
			Status:       driver3.Confirmed,
		},
		{
			TxID:         "7",
			ActionType:   driver3.Transfer,
			SenderEID:    "alice",
			RecipientEID: "dan",
			TokenType:    "abc",
			Amount:       big.NewInt(1),
			Timestamp:    now.AddDate(0, 0, -1),
			Status:       driver3.Confirmed,
		},
		{
			TxID:         "8",
			ActionType:   driver3.Redeem,
			SenderEID:    "dan",
			RecipientEID: "carlos",
			TokenType:    "abc",
			Amount:       big.NewInt(1),
			Timestamp:    now.AddDate(0, 0, -1),
			Status:       driver3.Confirmed,
		},
		{
			TxID:         "9",
			ActionType:   driver3.Transfer,
			SenderEID:    "alice",
			RecipientEID: "dan",
			TokenType:    "abc",
			Amount:       big.NewInt(1),
			Timestamp:    now.AddDate(0, 0, 1),
			Status:       driver3.Confirmed,
		},
		{
			TxID:         "10",
			ActionType:   driver3.Redeem,
			SenderEID:    "alice",
			RecipientEID: "",
			TokenType:    "abc",
			Amount:       big.NewInt(1),
			Timestamp:    now.AddDate(0, 0, 1),
			Status:       driver3.Confirmed,
		},
	}
	testCases := []struct {
		name        string
		params      driver3.QueryTransactionsParams
		expectedLen int
		expectedSql string
	}{
		{
			name:        "No params",
			params:      driver3.QueryTransactionsParams{},
			expectedLen: len(tr),
		},
		{
			name: "Only driver.Confirmed",
			params: driver3.QueryTransactionsParams{
				Statuses: []driver3.TxStatus{driver3.Confirmed},
			},
			expectedLen: 10,
		},
		{
			name: "Pending and deleted",
			params: driver3.QueryTransactionsParams{
				Statuses: []driver3.TxStatus{driver3.Pending, driver3.Deleted},
			},
			expectedLen: 2,
		},
		{
			name: "Confirmed from alice should return all driver.Confirmed",
			params: driver3.QueryTransactionsParams{
				SenderWallet: "alice",
				Statuses:     []driver3.TxStatus{driver3.Confirmed},
			},
			expectedLen: 10,
		},
		{
			name: "Recipient matches should return all",
			params: driver3.QueryTransactionsParams{
				RecipientWallet: "alice",
			},
			expectedLen: 12,
		},
		{
			name: "Sender OR recipient matches",
			params: driver3.QueryTransactionsParams{
				SenderWallet:    "alice",
				RecipientWallet: "alice",
			},
			expectedLen: 9,
		},
		{
			name: "Sender OR recipient matches, from last year",
			params: driver3.QueryTransactionsParams{
				SenderWallet:    "alice",
				RecipientWallet: "alice",
				From:            &lastYear,
			},
			expectedLen: 9,
		},
		{
			name: "Only this millisecond",
			params: driver3.QueryTransactionsParams{
				From: &justBefore,
				To:   &justAfter,
			},
			expectedLen: 7,
		},
		{
			name: "Only this millisecond for alice",
			params: driver3.QueryTransactionsParams{
				SenderWallet:    "alice",
				RecipientWallet: "alice",
				From:            &justBefore,
				To:              &justAfter,
			},
			expectedLen: 5,
		},
		{
			name: "Get redemption",
			params: driver3.QueryTransactionsParams{
				ActionTypes: []driver3.ActionType{driver3.Redeem},
			},
			expectedLen: 2,
		},
	}

	w, err := db.BeginAtomicWrite()
	assert.NoError(t, err)
	var previous string
	for _, r := range tr {
		if r.TxID != previous {
			assert.NoError(t, w.AddTokenRequest(ctx, r.TxID, []byte{}, map[string][]byte{}, nil, driver2.PPHash("tr")))
		}
		assert.NoError(t, w.AddTransaction(ctx, r))
		previous = r.TxID
	}
	assert.NoError(t, w.Commit())
	for _, r := range tr {
		if r.Status != driver3.Pending {
			assert.NoError(t, db.SetStatus(context.TODO(), r.TxID, r.Status, ""))
		}
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := getTransactions(t, db, tc.params)
			assert.Len(t, res, tc.expectedLen, fmt.Sprintf("params: %v", tc.params))
		})
	}
}

func getTransactions(t *testing.T, db driver3.TokenTransactionStore, params driver3.QueryTransactionsParams) []*driver3.TransactionRecord {
	records, err := db.QueryTransactions(context.Background(), params, pagination.None())
	assert.NoError(t, err)
	txs, err := iterators.ReadAllPointers(records.Items)
	assert.NoError(t, err)
	return txs
}

func TValidationRecordQueries(t *testing.T, db driver3.TokenTransactionStore) {
	beforeTx := time.Now().UTC().Add(-1 * time.Second)
	ctx := context.Background()
	exp := []driver3.ValidationRecord{
		{
			TxID:         "1",
			TokenRequest: []byte("tr1"),
			Metadata: map[string][]byte{
				"key": []byte("value"),
			},
			Status: driver3.Unknown,
		},
		{
			TxID:         "2",
			TokenRequest: []byte{},
			Metadata:     nil,
			Status:       driver3.Unknown,
		},
		{
			TxID:         "3",
			TokenRequest: []byte("tr3"),
			Metadata: map[string][]byte{
				"key": []byte("value"),
			},
			Status: driver3.Unknown,
		},
		{
			TxID:         "4",
			TokenRequest: []byte("tr4"),
			Metadata: map[string][]byte{
				"key": []byte("value"),
			},
			Status: driver3.Confirmed,
		},
	}
	w, err := db.BeginAtomicWrite()
	assert.NoError(t, err)
	for _, e := range exp {
		assert.NoError(t, w.AddTokenRequest(ctx, e.TxID, e.TokenRequest, map[string][]byte{}, nil, driver2.PPHash("tr")))
		assert.NoError(t, w.AddValidationRecord(ctx, e.TxID, e.Metadata), "AddValidationRecord "+e.TxID)
	}
	assert.NoError(t, w.Commit(), "Commit")
	for _, e := range exp {
		if e.Status != driver3.Pending {
			assert.NoError(t, db.SetStatus(context.TODO(), e.TxID, e.Status, ""))
		}
	}
	all := getValidationRecords(t, db, driver3.QueryValidationRecordsParams{})
	assert.Len(t, all, 4)

	for i, vr := range exp {
		assert.Equal(t, vr.TxID, all[i].TxID, fmt.Sprintf("%v", all[i]))
		assert.Equal(t, vr.TokenRequest, all[i].TokenRequest, fmt.Sprintf("%v - %d", all[i], len(all[i].TokenRequest)))
		assert.Equal(t, vr.Metadata, all[i].Metadata, fmt.Sprintf("%v", all[i]))
		assert.Equal(t, vr.Status, all[i].Status, fmt.Sprintf("%v", all[i]))
		assert.WithinDuration(t, beforeTx, all[i].Timestamp, 5*time.Second, fmt.Sprintf("%v", all[i]))
	}

	to := getValidationRecords(t, db, driver3.QueryValidationRecordsParams{
		To: &beforeTx,
	})
	assert.Len(t, to, 0, "Expect no results if all records are created after 'To'")

	from := getValidationRecords(t, db, driver3.QueryValidationRecordsParams{
		From: &beforeTx,
	})
	assert.Len(t, from, len(exp), "'From' before creation should include all records'")

	confirmed := getValidationRecords(t, db, driver3.QueryValidationRecordsParams{
		Statuses: []driver3.TxStatus{driver3.Confirmed},
	})
	assert.Len(t, confirmed, 1)

	filtered := getValidationRecords(t, db, driver3.QueryValidationRecordsParams{
		Filter: func(r *driver3.ValidationRecord) bool {
			return r.Status == driver3.Unknown
		},
	})
	assert.Len(t, filtered, 3)
}

func getValidationRecords(t *testing.T, db driver3.TokenTransactionStore, params driver3.QueryValidationRecordsParams) []*driver3.ValidationRecord {
	records, err := db.QueryValidations(context.Background(), params)
	assert.NoError(t, err)
	txs, err := iterators.ReadAllPointers(records)
	assert.NoError(t, err)
	return txs
}

func TEndorserAcks(t *testing.T, db driver3.TokenTransactionStore) {
	ctx := context.Background()
	createTestTransaction(t, db, "1")
	wg := sync.WaitGroup{}
	n := 100
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			assert.NoError(t, db.AddTransactionEndorsementAck(ctx, "1", []byte(fmt.Sprintf("alice_%d", i)), []byte(fmt.Sprintf("sigma_%d", i))))
			acks, err := db.GetTransactionEndorsementAcks(ctx, "1")
			assert.NoError(t, err)
			assert.True(t, len(acks) != 0)
			wg.Done()
		}(i)
	}
	wg.Wait()

	acks, err := db.GetTransactionEndorsementAcks(ctx, "1")
	assert.NoError(t, err)
	assert.Len(t, acks, n)
	for i := 0; i < n; i++ {
		assert.Equal(t, []byte(fmt.Sprintf("sigma_%d", i)), acks[token.Identity(fmt.Sprintf("alice_%d", i)).String()])
	}
}

func createTestTransaction(t *testing.T, db driver3.TokenTransactionStore, txID string) {
	w, err := db.BeginAtomicWrite()
	if err != nil {
		t.Fatalf("error creating transaction while trying to test something else: %s", err)
	}
	if err := w.AddTokenRequest(context.Background(), txID, []byte{}, map[string][]byte{}, nil, driver2.PPHash("tr")); err != nil {
		t.Fatalf("error creating token request while trying to test something else: %s", err)
	}
	tr1 := driver3.TransactionRecord{
		TxID:         txID,
		ActionType:   driver3.Transfer,
		SenderEID:    "bob",
		RecipientEID: "alice",
		TokenType:    "magic",
		Amount:       big.NewInt(10),
		Timestamp:    time.Now().Local().UTC(),
		Status:       driver3.Pending,
	}
	if err := w.AddTransaction(context.Background(), tr1); err != nil {
		t.Fatalf("error creating transaction while trying to test something else: %s", err)
	}
	if err := w.Commit(); err != nil {
		t.Fatalf("error committing transaction while trying to test something else: %s", err)
	}
}
