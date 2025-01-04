/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"fmt"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type TokenInterpreter interface {
	common.Interpreter
	HasTokens(colTxID, colIdx common.FieldName, ids ...*token.ID) common.Condition
	HasTokenTypes(colTokenType common.FieldName, tokenTypes ...string) common.Condition
	HasTokenDetails(params driver.QueryTokenDetailsParams, tokenTable string) common.Condition
	HasMovementsParams(params driver.QueryMovementsParams) common.Condition
	HasValidationParams(params driver.QueryValidationRecordsParams) common.Condition
	HasTransactionParams(params driver.QueryTransactionsParams, table string) common.Condition
}

func NewTokenInterpreter(ci common.Interpreter) TokenInterpreter {
	return &tokenInterpreter{Interpreter: ci}
}

type tokenInterpreter struct {
	common.Interpreter
}

func (c *tokenInterpreter) HasTokens(colTxID, colIdx common.FieldName, ids ...*token.ID) common.Condition {
	if len(ids) == 0 {
		return common.EmptyCondition
	}

	vals := make([]common.Tuple, len(ids))
	for i, id := range ids {
		vals[i] = common.Tuple{id.TxId, id.Index}
	}
	return c.InTuple([]common.FieldName{colTxID, colIdx}, vals)
}

func (c *tokenInterpreter) HasTokenTypes(colTokenType common.FieldName, tokenTypes ...string) common.Condition {
	if len(tokenTypes) == 0 {
		return common.EmptyCondition
	}
	return c.InStrings(colTokenType, tokenTypes)
}

func (c *tokenInterpreter) HasTokenDetails(params driver.QueryTokenDetailsParams, tokenTable string) common.Condition {
	conds := []common.Condition{
		common.ConstCondition("owner = true"),
		c.Cmp("owner_type", "=", params.OwnerType),
		c.Cmp("token_type", "=", params.TokenType),
		c.InStrings(common.JoinCol(tokenTable, "tx_id"), params.TransactionIDs),
		c.HasTokens(common.JoinCol(tokenTable, "tx_id"), common.JoinCol(tokenTable, "idx"), params.IDs...),
	}
	if len(tokenTable) > 0 {
		conds = append(conds, c.Or(c.Cmp("wallet_id", "=", params.WalletID), c.Cmp("owner_wallet_id", "=", params.WalletID)))
	} else {
		conds = append(conds, c.Cmp("owner_wallet_id", "=", params.WalletID))
	}
	if !params.IncludeDeleted {
		conds = append(conds, common.ConstCondition("is_deleted = false"))
	}
	if params.OnlyNonSpendable {
		conds = append(conds, common.ConstCondition("spendable = false"))
	}
	if params.OnlySpendable {
		conds = append(conds, common.ConstCondition("spendable = true"))
	}
	if len(params.LedgerTokenTypes) > 0 {
		types := make([]string, len(params.LedgerTokenTypes))
		for i, typ := range params.LedgerTokenTypes {
			types[i] = string(typ)
		}
		conds = append(conds, c.InStrings("ledger_type", types))
	}
	return c.And(conds...)
}

func (c *tokenInterpreter) HasMovementsParams(params driver.QueryMovementsParams) common.Condition {
	tokenTypes := make([]string, len(params.TokenTypes))
	for i, typ := range params.TokenTypes {
		tokenTypes[i] = string(typ)
	}
	conds := []common.Condition{
		c.InStrings("enrollment_id", params.EnrollmentIDs),
		c.InStrings("token_type", tokenTypes),
		c.InInts("status", params.TxStatuses),
	}

	if len(params.TxStatuses) == 0 {
		conds = append(conds, common.ConstCondition(fmt.Sprintf("status != %d", driver.Deleted)))
	}

	if params.MovementDirection == driver.Sent {
		conds = append(conds, common.ConstCondition("amount < 0"))
	} else if params.MovementDirection == driver.Received {
		conds = append(conds, common.ConstCondition("amount > 0"))
	}
	return c.And(conds...)
}

func (c *tokenInterpreter) HasValidationParams(params driver.QueryValidationRecordsParams) common.Condition {
	var conds []common.Condition

	if params.From != nil && !params.From.IsZero() {
		conds = append(conds, c.Cmp("stored_at", ">=", params.From.UTC()))
	}
	if params.To != nil && !params.To.IsZero() {
		conds = append(conds, c.Cmp("stored_at", "<=", params.To.UTC()))
	}
	if len(params.Statuses) > 0 {
		conds = append(conds, c.InInts("status", common.ToInts(params.Statuses)))
	}
	return c.And(conds...)
}

func (c *tokenInterpreter) HasTransactionParams(params driver.QueryTransactionsParams, table string) common.Condition {
	conds := []common.Condition{
		c.InStrings(common.JoinCol(table, "tx_id"), params.IDs),
	}

	if params.ExcludeToSelf {
		conds = append(conds, common.ConstCondition("sender_eid != recipient_eid"))
	}
	if params.From != nil && !params.From.IsZero() {
		conds = append(conds, c.Cmp("stored_at", ">=", params.From.UTC()))
	}
	if params.To != nil && !params.To.IsZero() {
		conds = append(conds, c.Cmp("stored_at", "<=", params.To.UTC()))
	}
	if len(params.ActionTypes) > 0 {
		conds = append(conds, c.InInts("action_type", common.ToInts(params.ActionTypes)))
	}
	// Specific transaction status if requested, defaults to all but Deleted
	if len(params.Statuses) > 0 {
		conds = append(conds, c.InInts("status", common.ToInts(params.Statuses)))
	}

	// See QueryTransactionsParams for expected behavior. If only one of sender or
	// recipient is set, we return all transactions. If both are set, we do an OR.
	if params.SenderWallet != "" && params.RecipientWallet != "" {
		conds = append(conds, c.Or(
			c.Cmp("sender_eid", "=", params.SenderWallet),
			c.Cmp("recipient_eid", "=", params.RecipientWallet),
		))
	}
	return c.And(conds...)
}
