/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"fmt"
	"strings"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

func transactionsConditionsSql(params driver.QueryTransactionsParams, table string) (where string, args []any) {
	and := []string{}

	// By id(s)
	if len(params.IDs) > 0 {
		colTxID := "tx_id"
		if len(table) > 0 {
			colTxID = fmt.Sprintf("%s.%s", table, colTxID)
		}
		and = append(and, in(&args, colTxID, params.IDs))
	}
	// Filter out 'change' transactions
	if params.ExcludeToSelf {
		and = append(and, "(sender_eid != recipient_eid)")
	}

	// Timestamp from
	if params.From != nil && !params.From.IsZero() {
		args = append(args, params.From.UTC())
		where := fmt.Sprintf("stored_at >= %s", fmt.Sprintf("$%d", len(args)))
		and = append(and, where)
	}

	// Timestamp to
	if params.To != nil && !params.To.IsZero() {
		args = append(args, params.To.UTC())
		where := fmt.Sprintf("stored_at <= %s", fmt.Sprintf("$%d", len(args)))
		and = append(and, where)
	}

	// Action types (issue, transfer or redeem)
	if len(params.ActionTypes) > 0 {
		and = append(and, in(&args, "action_type", params.ActionTypes))
	}

	// Specific transaction status if requested, defaults to all but Deleted
	if len(params.Statuses) > 0 {
		and = append(and, in(&args, "status", params.Statuses))
	}

	// See QueryTransactionsParams for expected behavior. If only one of sender or
	// recipient is set, we return all transactions. If both are set, we do an OR.
	if params.SenderWallet != "" && params.RecipientWallet != "" {
		args = append(args, params.SenderWallet, params.RecipientWallet)
		where := fmt.Sprintf("(sender_eid = %s OR recipient_eid = %s)", fmt.Sprintf("$%d", len(args)-1), fmt.Sprintf("$%d", len(args)))
		and = append(and, where)
	}

	if len(and) == 0 {
		return "", args
	}
	where = fmt.Sprintf("WHERE %s", strings.Join(and, " AND "))

	return
}

func validationConditionsSql(params driver.QueryValidationRecordsParams) (where string, args []any) {
	and := []string{}

	// Timestamp from
	if params.From != nil && !params.From.IsZero() {
		args = append(args, params.From)
		where := fmt.Sprintf("stored_at >= %s", fmt.Sprintf("$%d", len(args)))
		and = append(and, where)
	}

	// Timestamp to
	if params.To != nil && !params.To.IsZero() {
		args = append(args, params.To)
		where := fmt.Sprintf("stored_at <= %s", fmt.Sprintf("$%d", len(args)))
		and = append(and, where)
	}

	// Specific transaction status if requested, defaults to all but Deleted
	if len(params.Statuses) > 0 {
		and = append(and, in(&args, "status", params.Statuses))
	}
	if len(and) > 0 {
		where = fmt.Sprintf("WHERE %s", strings.Join(and, " AND "))
	}

	return
}

func movementConditionsSql(params driver.QueryMovementsParams) (where string, args []any) {
	and := []string{}

	// Filter by enrollment id (any)
	if len(params.EnrollmentIDs) > 0 {
		and = append(and, in(&args, "enrollment_id", params.EnrollmentIDs))
	}

	// Filter by token type (any)
	if len(params.TokenTypes) > 0 {
		and = append(and, in(&args, "token_type", params.TokenTypes))
	}

	// Specific transaction status if requested, defaults to all but Deleted
	if len(params.TxStatuses) > 0 {
		and = append(and, in(&args, "status", params.TxStatuses))
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

	where = fmt.Sprintf("WHERE %s%s%s", strings.Join(and, " AND "), order, limit)

	return
}

func tokenRequestConditionsSql(params driver.QueryTokenRequestsParams) (string, []any) {
	args := make([]any, 0)
	and := make([]string, 0)
	if len(params.Statuses) == 0 {
		return "", args
	}

	// Specific transaction status if requested, defaults to all but Deleted
	if len(params.Statuses) > 0 {
		and = append(and, in(&args, "status", params.Statuses))
	}
	where := fmt.Sprintf("WHERE %s", strings.Join(and, " AND "))

	return where, args
}

func in[T string | driver.TxStatus | driver.ActionType](args *[]any, field string, searchFor []T) (where string) {
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

// tokenQuerySql requires a join with the token ownership table if OwnerEnrollmentID is not empty
func tokenQuerySql(params driver.QueryTokenDetailsParams, tokenTable, ownerTable string) (where, join string, args []any) {
	and := []string{"owner = true"}
	if params.OwnerType != "" {
		args = append(args, params.OwnerType)
		and = append(and, fmt.Sprintf("owner_type = $%d", len(args)))
	}
	if params.OwnerEnrollmentID != "" {
		args = append(args, params.OwnerEnrollmentID)
		and = append(and, fmt.Sprintf("enrollment_id = $%d", len(args)))
	}

	if params.TokenType != "" {
		args = append(args, params.TokenType)
		and = append(and, fmt.Sprintf("token_type = $%d", len(args)))
	}

	if len(params.TransactionIDs) > 0 {
		colTxID := "tx_id"
		if len(tokenTable) > 0 {
			colTxID = fmt.Sprintf("%s.%s", tokenTable, colTxID)
		}
		and = append(and, in(&args, colTxID, params.TransactionIDs))
	}
	if ids := whereTokenIDsForJoin(tokenTable, &args, params.IDs); ids != "" {
		and = append(and, ids)
	}

	if !params.IncludeDeleted {
		and = append(and, "is_deleted = false")
	}

	join = joinOnTokenID(tokenTable, ownerTable)
	where = fmt.Sprintf("WHERE %s", strings.Join(and, " AND "))

	return
}

func whereTokenIDsForJoin(tableName string, args *[]any, ids []*token.ID) (where string) {
	if len(ids) == 0 {
		return ""
	}

	colTxID := "tx_id"
	colIdx := "idx"
	if len(tableName) > 0 {
		colTxID = fmt.Sprintf("%s.%s", tableName, colTxID)
		colIdx = fmt.Sprintf("%s.%s", tableName, colIdx)
	}

	in := make([]string, len(ids))
	for i, id := range ids {
		*args = append(*args, id.TxId, id.Index)
		in[i] = fmt.Sprintf("($%d, $%d)", len(*args)-1, len(*args))
	}
	return fmt.Sprintf("(%s, %s) IN ( %s )", colTxID, colIdx, strings.Join(in, ", "))
}

func whereTokenIDs(args *[]any, ids []*token.ID) (where string) {
	return whereTokenIDsForJoin("", args, ids)
}

func joinOnTxID(table, other string) string {
	return fmt.Sprintf("LEFT JOIN %s ON %s.tx_id = %s.tx_id", other, table, other)
}

func joinOnTokenID(table, other string) string {
	return fmt.Sprintf("LEFT JOIN %s ON %s.tx_id = %s.tx_id AND %s.idx = %s.idx", other, table, other, table, other)
}
