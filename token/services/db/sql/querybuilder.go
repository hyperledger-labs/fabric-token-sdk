/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"fmt"
	"strings"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
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
			t[i] = s
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
		return "", args
	}
	where := fmt.Sprintf("WHERE %s", strings.Join(and, " AND "))

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
			t[i] = int(s)
		}
		add(&and, in(&args, "status", t))
	}

	if len(and) == 0 {
		return "", args
	}
	where := fmt.Sprintf("WHERE %s", strings.Join(and, " AND "))

	return where, args
}

func movementConditionsSql(params driver.QueryMovementsParams) (string, []interface{}) {
	args := make([]interface{}, 0)
	and := make([]string, 0)

	// Filter by enrollment id (any)
	t := make([]interface{}, len(params.EnrollmentIDs))
	for i, s := range params.EnrollmentIDs {
		t[i] = s
	}
	add(&and, in(&args, "enrollment_id", t))

	// Filter by token type (any)
	t = make([]interface{}, len(params.TokenTypes))
	for i, s := range params.TokenTypes {
		t[i] = s
	}
	add(&and, in(&args, "token_type", t))

	// Specific transaction status if requested, defaults to all but Deleted
	if len(params.TxStatuses) > 0 {
		statuses := make([]interface{}, len(params.TxStatuses))
		for i, s := range params.TxStatuses {
			statuses[i] = s
		}
		add(&and, in(&args, "status", statuses))
	} else {
		and = append(and, fmt.Sprintf("status != %d", driver.Deleted))
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

	where := fmt.Sprintf("WHERE %s%s%s", strings.Join(and, " AND "), order, limit)

	return where, args
}

func certificationsQuerySql(ids []*token.ID) (string, []any, error) {
	if len(ids) == 0 {
		return "", nil, nil
	}
	if ids[0] == nil {
		return "", nil, errors.Errorf("invalid token-id, cannot be nil")
	}
	var builder strings.Builder
	builder.WriteString("token_id=$1")
	var tokenIDs []any
	tokenIDs = []any{fmt.Sprintf("%s%d", ids[0].TxId, ids[0].Index)}
	for i := 1; i <= len(ids)-1; i++ {
		if ids[i] == nil {
			return "", nil, errors.Errorf("invalid token-id, cannot be nil")
		}
		builder.WriteString(" || ")
		builder.WriteString(fmt.Sprintf("token_id=$%d", i+1))
		tokenIDs = append(tokenIDs, fmt.Sprintf("%s%d", ids[i].TxId, ids[i].Index))
	}
	builder.WriteString("")

	return builder.String(), tokenIDs, nil
}

func tokenRequestConditionsSql(params driver.QueryTokenRequestsParams) (string, []interface{}) {
	args := make([]interface{}, 0)
	and := make([]string, 0)

	// Specific transaction status if requested, defaults to all but Deleted
	if len(params.Statuses) > 0 {
		t := make([]interface{}, len(params.Statuses))
		for i, s := range params.Statuses {
			t[i] = int(s)
		}
		add(&and, in(&args, "status", t))
	}

	if len(and) == 0 {
		return "", args
	}
	where := fmt.Sprintf("WHERE %s", strings.Join(and, " AND "))

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

func whereTokenIDs(args *[]interface{}, ids []*token.ID) string {
	switch len(ids) {
	case 0:
		return ""
	case 1:
		*args = append(*args, ids[0].TxId, ids[0].Index)
		l := len(*args)
		return fmt.Sprintf("tx_id = %s AND idx = %s", fmt.Sprintf("$%d", l-1), fmt.Sprintf("$%d", l))
	default:
		in := make([]string, len(ids))
		for i, id := range ids {
			*args = append(*args, id.TxId, id.Index)
			l := len(*args)
			in[i] = fmt.Sprintf("($%d, $%d)", l-1, l)
		}
		return fmt.Sprintf("(tx_id, idx) IN ( %s )", strings.Join(in, ", "))
	}
}

func add(and *[]string, clause string) {
	if clause != "" {
		*and = append(*and, clause)
	}
}

func joinOnTxID(table, parent string) string {
	return fmt.Sprintf("LEFT JOIN %s ON %s.tx_id = %s.tx_id", parent, table, parent)
}
