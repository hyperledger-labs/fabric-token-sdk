/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"fmt"
	"strings"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb/driver"
)

func transactionsConditionsSql(params driver.QueryTransactionsParams) (string, []interface{}) {
	args := make([]interface{}, 0)
	and := make([]string, 0)

	// Timestamp from
	if params.From != nil && !params.From.IsZero() {
		where := fmt.Sprintf("stored_at >= %s", fmt.Sprintf("$%d", len(args)+1))
		args = append(args, params.From.UTC())
		and = append(and, where)
	}

	// Timestamp to
	if params.To != nil && !params.To.IsZero() {
		where := fmt.Sprintf("stored_at <= %s", fmt.Sprintf("$%d", len(args)+1))
		args = append(args, params.To.UTC())
		and = append(and, where)
	}

	// Action types
	if len(params.ActionTypes) > 0 {
		t := make([]interface{}, len(params.ActionTypes))
		for i, s := range params.ActionTypes {
			t[i] = int(s)
		}
		add(&and, in(&args, "action_type", t))
	}

	// Specific transaction status if requested, defaults to all but Deleted
	if len(params.Statuses) > 0 {
		t := make([]interface{}, len(params.Statuses))
		for i, s := range params.Statuses {
			t[i] = string(s)
		}
		add(&and, in(&args, "status", t))
	}

	// See QueryTransactionsParams for expected behavior. If only one of sender or
	// recipient is set, we return all transactions. If both are set, we do an OR.
	if params.SenderWallet != "" && params.RecipientWallet != "" {
		where := fmt.Sprintf("(sender_eid = %s OR recipient_eid = %s)", fmt.Sprintf("$%d", len(args)+1), fmt.Sprintf("$%d", len(args)+2))
		args = append(args, params.SenderWallet, params.RecipientWallet)
		and = append(and, where)
	}

	if len(and) == 0 {
		return ";", args
	}
	where := fmt.Sprintf("WHERE %s;", strings.Join(and, " AND "))

	return where, args
}

func validationConditionsSql(params driver.QueryValidationRecordsParams) (string, []interface{}) {
	args := make([]interface{}, 0)
	and := make([]string, 0)

	// Timestamp from
	if params.From != nil && !params.From.IsZero() {
		where := fmt.Sprintf("stored_at >= %s", fmt.Sprintf("$%d", len(args)+1))
		args = append(args, params.From)
		and = append(and, where)
	}

	// Timestamp to
	if params.To != nil && !params.To.IsZero() {
		where := fmt.Sprintf("stored_at <= %s", fmt.Sprintf("$%d", len(args)+1))
		args = append(args, params.To)
		and = append(and, where)
	}

	// Specific transaction status if requested, defaults to all but Deleted
	if len(params.Statuses) > 0 {
		t := make([]interface{}, len(params.Statuses))
		for i, s := range params.Statuses {
			t[i] = string(s)
		}
		add(&and, in(&args, "status", t))
	}

	if len(and) == 0 {
		return ";", args
	}
	where := fmt.Sprintf("WHERE %s;", strings.Join(and, " AND "))

	return where, args
}

func movementConditionsSql(params driver.QueryMovementsParams) (string, []interface{}) {
	args := make([]interface{}, 0)
	and := make([]string, 0)

	// Filter by enrollment id (any)
	t := make([]interface{}, len(params.EnrollmentIDs))
	for i, s := range params.EnrollmentIDs {
		t[i] = string(s)
	}
	add(&and, in(&args, "enrollment_id", t))

	// Filter by token type (any)
	t = make([]interface{}, len(params.TokenTypes))
	for i, s := range params.TokenTypes {
		t[i] = string(s)
	}
	add(&and, in(&args, "token_type", t))

	// Specific transaction status if requested, defaults to all but Deleted
	if len(params.TxStatuses) > 0 {
		statuses := make([]interface{}, len(params.TxStatuses))
		for i, s := range params.TxStatuses {
			statuses[i] = string(s)
		}
		add(&and, in(&args, "status", statuses))
	} else {
		and = append(and, "status != 'Deleted'")
	}

	// Sent or received
	if params.MovementDirection == driver.Sent {
		and = append(and, "amount < 0")
	}
	if params.MovementDirection == driver.Received {
		and = append(and, "amount > 0")
	}

	// Order by stored_at
	order := ""
	if params.SearchDirection == driver.FromBeginning {
		order = " ORDER BY stored_at ASC"
	} else {
		order = " ORDER BY stored_at DESC"
	}

	// Limit number of results
	limit := ""
	if params.NumRecords != 0 {
		limit = fmt.Sprintf(" LIMIT %d", params.NumRecords)
	}

	where := fmt.Sprintf("WHERE %s%s%s;", strings.Join(and, " AND "), order, limit)

	return where, args
}

func in(args *[]interface{}, field string, searchFor []interface{}) (where string) {
	if len(searchFor) == 0 {
		return ""
	}

	argnum := make([]string, len(searchFor))
	start := len(*args)
	for i, eid := range searchFor {
		argnum[i] = fmt.Sprintf("%s = %s", field, fmt.Sprintf("$%d", start+i+1))
		*args = append(*args, eid)
	}
	if len(argnum) == 1 {
		return argnum[0]
	}

	return fmt.Sprintf("(%s)", strings.Join(argnum, " OR "))
}

func add(and *[]string, clause string) {
	if clause != "" {
		*and = append(*and, clause)
	}
}
