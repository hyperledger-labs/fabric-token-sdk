/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"fmt"
	"testing"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/test-go/testify/assert"
)

func TestTransactionSql(t *testing.T) {
	now := time.Now().Local().UTC()
	lastYear := now.AddDate(-1, 0, 0)
	testCases := []struct {
		name         string
		params       driver.QueryTransactionsParams
		expectedArgs []interface{}
		expectedSql  string
	}{
		{
			name:         "No params",
			params:       driver.QueryTransactionsParams{},
			expectedSql:  "",
			expectedArgs: []interface{}{},
		},
		{
			name: "Only confirmed",
			params: driver.QueryTransactionsParams{
				Statuses: []driver.TxStatus{driver.Confirmed},
			},
			expectedSql:  "WHERE status = $1",
			expectedArgs: []interface{}{driver.Confirmed},
		},
		{
			name: "Pending or deleted",
			params: driver.QueryTransactionsParams{
				Statuses: []driver.TxStatus{driver.Pending, driver.Deleted},
			},
			expectedSql:  "WHERE (status = $1 OR status = $2)",
			expectedArgs: []interface{}{driver.Pending, driver.Deleted},
		},
		{
			name: "Confirmed from any (only setting sender should return all)",
			params: driver.QueryTransactionsParams{
				SenderWallet: "alice",
				Statuses:     []driver.TxStatus{driver.Confirmed},
			},
			expectedSql:  "WHERE status = $1",
			expectedArgs: []interface{}{driver.Confirmed},
		},
		{
			name: "Sender OR recipient matches",
			params: driver.QueryTransactionsParams{
				SenderWallet:    "alice",
				RecipientWallet: "bob",
			},
			expectedSql:  "WHERE (sender_eid = $1 OR recipient_eid = $2)",
			expectedArgs: []interface{}{"alice", "bob"},
		},
		{
			name: "Sender OR recipient matches, from last year",
			params: driver.QueryTransactionsParams{
				SenderWallet:    "alice",
				RecipientWallet: "alice",
				From:            &lastYear,
			},
			expectedSql:  "WHERE stored_at >= $1 AND (sender_eid = $2 OR recipient_eid = $3)",
			expectedArgs: []interface{}{&lastYear, "alice", "alice"},
		},
		{
			name: "From last year to now",
			params: driver.QueryTransactionsParams{
				To:   &now,
				From: &lastYear,
			},
			expectedSql:  "WHERE stored_at >= $1 AND stored_at <= $2",
			expectedArgs: []interface{}{&lastYear, &now},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualSql, actualArgs := transactionsConditionsSql(tc.params)
			assert.Equal(t, tc.expectedSql, actualSql)
			compareArgs(t, tc.expectedArgs, actualArgs)
		})
	}
}

func TestMovementConditions(t *testing.T) {
	testCases := []struct {
		name         string
		params       driver.QueryMovementsParams
		expectedArgs []interface{}
		expectedSql  string
	}{
		{
			name: "All",
			params: driver.QueryMovementsParams{
				MovementDirection: driver.All,
			},
			expectedSql:  "WHERE status != 3 ORDER BY stored_at DESC",
			expectedArgs: []interface{}{},
		},
		{
			name: "Max 5",
			params: driver.QueryMovementsParams{
				NumRecords:        5,
				MovementDirection: driver.All,
			},
			expectedSql:  "WHERE status != 3 ORDER BY stored_at DESC LIMIT 5",
			expectedArgs: []interface{}{},
		},
		{
			name: "Only enrollment ids",
			params: driver.QueryMovementsParams{
				EnrollmentIDs:     []string{"eid1", "eid2", "eid3"},
				MovementDirection: driver.All,
			},
			expectedSql:  "WHERE (enrollment_id = $1 OR enrollment_id = $2 OR enrollment_id = $3) AND status != 3 ORDER BY stored_at DESC",
			expectedArgs: []interface{}{"eid1", "eid2", "eid3"},
		},
		{
			name: "Only confirmed",
			params: driver.QueryMovementsParams{
				TxStatuses:        []driver.TxStatus{driver.Confirmed},
				MovementDirection: driver.All,
			},
			expectedSql:  "WHERE status = $1 ORDER BY stored_at DESC",
			expectedArgs: []interface{}{driver.Confirmed},
		},
		{
			name: "Pending and deleted",
			params: driver.QueryMovementsParams{
				TxStatuses:        []driver.TxStatus{driver.Pending, driver.Deleted},
				MovementDirection: driver.All,
			},
			expectedSql:  "WHERE (status = $1 OR status = $2) ORDER BY stored_at DESC",
			expectedArgs: []interface{}{driver.Pending, driver.Deleted},
		},
		{
			name: "Confirmed from alice",
			params: driver.QueryMovementsParams{
				EnrollmentIDs:     []string{"alice"},
				TxStatuses:        []driver.TxStatus{driver.Confirmed},
				MovementDirection: driver.All,
			},
			expectedSql:  "WHERE enrollment_id = $1 AND status = $2 ORDER BY stored_at DESC",
			expectedArgs: []interface{}{"alice", driver.Confirmed},
		},
		{
			name: "Confirmed ABC and XYZ from alice",
			params: driver.QueryMovementsParams{
				EnrollmentIDs:     []string{"alice"},
				TxStatuses:        []driver.TxStatus{driver.Confirmed},
				TokenTypes:        []string{"ABC", "XYZ"},
				MovementDirection: driver.All,
			},
			expectedSql:  "WHERE enrollment_id = $1 AND (token_type = $2 OR token_type = $3) AND status = $4 ORDER BY stored_at DESC",
			expectedArgs: []interface{}{"alice", "ABC", "XYZ", driver.Confirmed},
		},
		{
			name: "Max 5 confirmed ABC and XYZ from alice",
			params: driver.QueryMovementsParams{
				EnrollmentIDs:     []string{"alice"},
				TxStatuses:        []driver.TxStatus{driver.Confirmed},
				TokenTypes:        []string{"ABC", "XYZ"},
				NumRecords:        5,
				MovementDirection: driver.All,
			},
			expectedSql:  "WHERE enrollment_id = $1 AND (token_type = $2 OR token_type = $3) AND status = $4 ORDER BY stored_at DESC LIMIT 5",
			expectedArgs: []interface{}{"alice", "ABC", "XYZ", driver.Confirmed},
		},
		{
			name: "Sent XYZ from alice",
			params: driver.QueryMovementsParams{
				EnrollmentIDs:     []string{"alice"},
				TokenTypes:        []string{"XYZ"},
				MovementDirection: driver.Sent,
			},
			expectedSql:  "WHERE enrollment_id = $1 AND token_type = $2 AND status != 3 AND amount < 0 ORDER BY stored_at DESC",
			expectedArgs: []interface{}{"alice", "XYZ"},
		},
		{
			name: "2 last pending received",
			params: driver.QueryMovementsParams{
				TxStatuses:        []driver.TxStatus{driver.Pending},
				SearchDirection:   driver.FromLast,
				MovementDirection: driver.Received,
				NumRecords:        2,
			},
			expectedSql:  "WHERE status = $1 AND amount > 0 ORDER BY stored_at DESC LIMIT 2",
			expectedArgs: []interface{}{driver.Pending},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualSql, actualArgs := movementConditionsSql(tc.params)
			assert.Equal(t, tc.expectedSql, actualSql)
			compareArgs(t, tc.expectedArgs, actualArgs)
		})
	}
}

func TestTokenSql(t *testing.T) {
	testCases := []struct {
		name         string
		ownerEID     string
		params       driver.QueryTokenDetailsParams
		expectedArgs []interface{}
		expectedSql  string
	}{
		{
			name:         "no filter",
			params:       driver.QueryTokenDetailsParams{},
			expectedSql:  "WHERE owner = true AND is_deleted = false",
			expectedArgs: []interface{}{},
		},
		{
			name: "no filter with deleted",
			params: driver.QueryTokenDetailsParams{
				IncludeDeleted: true,
			},
			expectedSql:  "WHERE owner = true",
			expectedArgs: []interface{}{},
		},
		{
			name:         "owner unspent",
			ownerEID:     "me",
			params:       driver.QueryTokenDetailsParams{},
			expectedSql:  "WHERE owner = true AND enrollment_id = $1 AND is_deleted = false",
			expectedArgs: []interface{}{"me"},
		},
		{
			name:     "owner with deleted",
			ownerEID: "me",
			params: driver.QueryTokenDetailsParams{
				IncludeDeleted: true,
			},
			expectedSql:  "WHERE owner = true AND enrollment_id = $1",
			expectedArgs: []interface{}{"me"},
		},
		{
			name:         "owner and type",
			ownerEID:     "me",
			params:       driver.QueryTokenDetailsParams{TokenType: "tok"},
			expectedSql:  "WHERE owner = true AND enrollment_id = $1 AND token_type = $2 AND is_deleted = false",
			expectedArgs: []interface{}{"me", "tok"},
		},
		{
			name:     "owner and type and id",
			ownerEID: "me",
			params: driver.QueryTokenDetailsParams{
				TokenType: "tok",
				IDs:       []*token.ID{{TxId: "a", Index: 1}},
			},
			expectedSql:  "WHERE owner = true AND enrollment_id = $1 AND token_type = $2 AND (tx_id, idx) IN ( ($3, $4) ) AND is_deleted = false",
			expectedArgs: []interface{}{"me", "tok", "a", 1},
		},
		{
			name: "type and ids",
			params: driver.QueryTokenDetailsParams{
				TokenType:      "tok",
				IDs:            []*token.ID{{TxId: "a", Index: 1}, {TxId: "b", Index: 2}},
				IncludeDeleted: true,
			},
			expectedSql:  "WHERE owner = true AND token_type = $1 AND (tx_id, idx) IN ( ($2, $3), ($4, $5) )",
			expectedArgs: []interface{}{"tok", "a", uint64(1), "b", uint64(2)},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualSql, _, actualArgs := tokenQuerySql(tc.ownerEID, tc.params, "", "")
			assert.Equal(t, tc.expectedSql, actualSql, tc.name)
			compareArgs(t, tc.expectedArgs, actualArgs)
		})
	}
	// with join
	where, join, args := tokenQuerySql("me", driver.QueryTokenDetailsParams{IDs: []*token.ID{{TxId: "a", Index: 1}}}, "A", "B")
	assert.Equal(t, "WHERE owner = true AND enrollment_id = $1 AND (A.tx_id, A.idx) IN ( ($2, $3) ) AND is_deleted = false", where, "join")
	assert.Equal(t, "LEFT JOIN B ON A.tx_id = B.tx_id AND A.idx = B.idx", join, "join")
	assert.Len(t, args, 3)
}

func TestIn(t *testing.T) {
	// 0
	args := make([]any, 0)
	w := in(&args, "enrollment_id", []string{})
	assert.Equal(t, "", w)
	assert.Equal(t, []any{}, args)

	// 1
	args = make([]any, 0)
	w = in(&args, "enrollment_id", []string{"eid1"})
	assert.Equal(t, "enrollment_id = $1", w)
	assert.Equal(t, []any{"eid1"}, args)

	// 3
	args = make([]any, 0)
	w = in(&args, "enrollment_id", []string{"eid1", "eid2", "eid3"})
	assert.Equal(t, "(enrollment_id = $1 OR enrollment_id = $2 OR enrollment_id = $3)", w)
	assert.Equal(t, []any{"eid1", "eid2", "eid3"}, args)
}

func compareArgs(t *testing.T, expected, actual []any) {
	assert.Len(t, actual, len(expected))
	// assert.Equal(t, tc.expectedArgs, actualArgs)

	for i := range expected {
		switch expected[i].(type) {
		case *time.Time:
			exp, _ := expected[i].(*time.Time)
			act, _ := actual[i].(time.Time)
			assert.True(t, exp.Equal(act), fmt.Sprintf("timestamps not equal: %v != %v", exp, act))
		default:
			assert.EqualValues(t, expected[i], actual[i])
		}
	}
}

func TestJoin(t *testing.T) {
	j := joinOnTxID("t1", "t2")
	assert.Equal(t, "LEFT JOIN t2 ON t1.tx_id = t2.tx_id", j)

	j = joinOnTokenID("t1", "t2")
	assert.Equal(t, "LEFT JOIN t2 ON t1.tx_id = t2.tx_id AND t1.idx = t2.idx", j)
}
