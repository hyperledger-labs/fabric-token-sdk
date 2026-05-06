/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/query/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/query/cond"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

func HasTokens(colTxID, colIdx string, ids ...*token.ID) sq.Sqlizer {
	if len(ids) == 0 {
		return sq.Expr("1=1")
	}
	or := make(sq.Or, len(ids))
	for i, id := range ids {
		or[i] = sq.And{sq.Eq{colTxID: id.TxId}, sq.Eq{colIdx: id.Index}}
	}
	return or
}

func HasTokenDetails(params driver2.QueryTokenDetailsParams, tableAlias string) sq.Sqlizer {
	var conds []sq.Sqlizer
	conds = append(conds, sq.Eq{"owner": true})

	if len(params.OwnerType) > 0 {
		conds = append(conds, sq.Eq{"owner_type": params.OwnerType})
	}
	if len(params.TokenType) > 0 {
		conds = append(conds, sq.Eq{"token_type": params.TokenType})
	}

	effectiveWallets := params.WalletIDs
	if len(effectiveWallets) == 0 && len(params.WalletID) > 0 {
		effectiveWallets = []string{params.WalletID}
	}

	if tableAlias != "" {
		// JOIN mode - wallet_id on Ownership, owner_wallet_id on Tokens
		if len(effectiveWallets) > 0 {
			conds = append(conds, sq.Or{
				sq.Eq{"wallet_id": effectiveWallets},
				sq.Eq{"owner_wallet_id": effectiveWallets},
			})
		}
		if len(params.TransactionIDs) > 0 {
			conds = append(conds, sq.Eq{tableAlias + ".tx_id": params.TransactionIDs})
		}
		conds = append(conds, HasTokens(tableAlias+".tx_id", tableAlias+".idx", params.IDs...))
	} else {
		if len(effectiveWallets) > 0 {
			conds = append(conds, sq.Eq{"owner_wallet_id": effectiveWallets})
		}
		if len(params.TransactionIDs) > 0 {
			conds = append(conds, sq.Eq{"tx_id": params.TransactionIDs})
		}
		conds = append(conds, HasTokens("tx_id", "idx", params.IDs...))
	}

	if !params.IncludeDeleted {
		conds = append(conds, sq.Eq{"is_deleted": false})
	}
	switch params.Spendable {
	case driver2.NonSpendableOnly:
		conds = append(conds, sq.Eq{"spendable": false})
	case driver2.SpendableOnly:
		conds = append(conds, sq.Eq{"spendable": true})
	}
	if len(params.LedgerTokenFormats) > 0 {
		conds = append(conds, sq.Eq{"ledger_type": params.LedgerTokenFormats})
	}

	return sq.And(conds)
}

func HasMovementsParams(params driver2.QueryMovementsParams) cond.Condition {
	conds := []cond.Condition{
		cond.In("enrollment_id", params.EnrollmentIDs...),
		cond.In("token_type", params.TokenTypes...),
		cond.In("status", params.TxStatuses...),
	}

	if len(params.TxStatuses) == 0 {
		conds = append(conds, cond.Neq("status", driver2.Deleted))
	}

	switch params.MovementDirection {
	case driver2.Sent:
		conds = append(conds, cond.Lt("amount", 0))
	case driver2.Received:
		conds = append(conds, cond.Gt("amount", 0))
	}

	return cond.And(conds...)
}

func HasValidationParams(params driver2.QueryValidationRecordsParams, tableName string) cond.Condition {
	return cond.And(
		cond.BetweenTimestamps(common.FieldName(tableName+".stored_at"), utc(params.From), utc(params.To)),
		cond.In("status", params.Statuses...),
	)
}

func HasTransactionParams(params driver2.QueryTransactionsParams, table common.Table) cond.Condition {
	conds := []cond.Condition{
		cond.FieldIn(table.Field("tx_id"), params.IDs...),
		cond.FieldBetweenTimestamps(table.Field("stored_at"), utc(params.From), utc(params.To)),
		cond.In("action_type", params.ActionTypes...),
		// Specific transaction status if requested, defaults to all but Deleted
		cond.In("status", params.Statuses...),
		cond.In("token_type", params.TokenTypes...),
	}

	if params.ExcludeToSelf {
		conds = append(conds, cond.Cmp(common.FieldName("sender_eid"), "!=", common.FieldName("recipient_eid")))
	}

	// See QueryTransactionsParams for expected behavior. If only one of sender or
	// recipient is set, we return all transactions. If both are set, we do an OR.
	if params.SenderWallet != "" && params.RecipientWallet != "" {
		conds = append(conds, cond.Or(
			cond.Eq("sender_eid", params.SenderWallet),
			cond.Eq("recipient_eid", params.RecipientWallet),
		))
	}

	return cond.And(conds...)
}

func utc(t *time.Time) time.Time {
	if t == nil || t.IsZero() {
		return time.Time{}
	}

	return t.UTC()
}
