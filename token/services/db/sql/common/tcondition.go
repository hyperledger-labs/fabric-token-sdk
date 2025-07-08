/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/query/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/query/cond"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

func HasTokens(colTxID, colIdx common.FieldName, ids ...*token.ID) cond.Condition {
	return hasTokens(colTxID, colIdx, ids...)
}

func hasTokens(colTxID, colIdx common.Field, ids ...*token.ID) cond.Condition {
	if len(ids) == 0 {
		return cond.AlwaysTrue
	}

	vals := make([]common.Tuple, len(ids))
	for i, id := range ids {
		vals[i] = common.Tuple{id.TxId, id.Index}
	}
	return cond.InTuple([]common.Serializable{colTxID, colIdx}, vals)
}

func HasTokenDetails(params driver.QueryTokenDetailsParams, tokenTable common.Table) cond.Condition {
	conds := []cond.Condition{cond.Eq("owner", true)}

	if len(params.OwnerType) > 0 {
		conds = append(conds, cond.Eq("owner_type", params.OwnerType))
	}
	if len(params.TokenType) > 0 {
		conds = append(conds, cond.Eq("token_type", params.TokenType))
	}
	if tokenTable != nil {
		if len(params.WalletID) > 0 {
			conds = append(conds, cond.Or(cond.Eq("wallet_id", params.WalletID), cond.Eq("owner_wallet_id", params.WalletID)))
		}
		conds = append(conds,
			cond.FieldIn(tokenTable.Field("tx_id"), params.TransactionIDs...),
			hasTokens(tokenTable.Field("tx_id"), tokenTable.Field("idx"), params.IDs...),
		)
	} else {
		if len(params.WalletID) > 0 {
			conds = append(conds, cond.Eq("owner_wallet_id", params.WalletID))
		}
		conds = append(conds,
			cond.FieldIn(common.FieldName("tx_id"), params.TransactionIDs...),
			hasTokens(common.FieldName("tx_id"), common.FieldName("idx"), params.IDs...),
		)
	}
	if !params.IncludeDeleted {
		conds = append(conds, cond.Eq("is_deleted", false))
	}
	switch params.Spendable {
	case driver.NonSpendableOnly:
		conds = append(conds, cond.Eq("spendable", false))
	case driver.SpendableOnly:
		conds = append(conds, cond.Eq("spendable", true))
	}

	conds = append(conds, cond.In("ledger_type", params.LedgerTokenFormats...))

	return cond.And(conds...)
}

func HasMovementsParams(params driver.QueryMovementsParams) cond.Condition {
	conds := []cond.Condition{
		cond.In("enrollment_id", params.EnrollmentIDs...),
		cond.In("token_type", params.TokenTypes...),
		cond.In("status", params.TxStatuses...),
	}

	if len(params.TxStatuses) == 0 {
		conds = append(conds, cond.Neq("status", driver.Deleted))
	}

	switch params.MovementDirection {
	case driver.Sent:
		conds = append(conds, cond.Lt("amount", 0))
	case driver.Received:
		conds = append(conds, cond.Gt("amount", 0))
	}
	return cond.And(conds...)
}

func HasValidationParams(params driver.QueryValidationRecordsParams) cond.Condition {
	return cond.And(
		cond.BetweenTimestamps("stored_at", utc(params.From), utc(params.To)),
		cond.In("status", params.Statuses...),
	)
}

func HasTransactionParams(params driver.QueryTransactionsParams, table common.Table) cond.Condition {
	conds := []cond.Condition{
		cond.FieldIn(table.Field("tx_id"), params.IDs...),
		cond.BetweenTimestamps("stored_at", utc(params.From), utc(params.To)),
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
