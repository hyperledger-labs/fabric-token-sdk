/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
)

var b = newTokenInterpreter()

func movementConditionsSql(params driver.QueryMovementsParams) (string, []any) {
	sb := strings.Builder{}

	where, args := common.Where(b.HasMovementsParams(params))
	sb.WriteString(where)

	// Order by stored_at
	if params.SearchDirection == driver.FromBeginning {
		sb.WriteString(" ORDER BY stored_at ASC")
	} else {
		sb.WriteString(" ORDER BY stored_at DESC")
	}

	// Limit number of results
	if params.NumRecords != 0 {
		sb.WriteString(" LIMIT ")
		sb.WriteString(strconv.Itoa(params.NumRecords))
	}

	return sb.String(), args
}

// tokenQuerySql requires a join with the token ownership table if WalletID is not empty
func tokenQuerySql(params driver.QueryTokenDetailsParams, tokenTable, ownerTable string) (string, string, []any) {
	w, ps := common.Where(b.HasTokenDetails(params, tokenTable))

	return w, joinOnTokenID(tokenTable, ownerTable), ps
}

func joinOnTxID(table, other string) string {
	return fmt.Sprintf("LEFT JOIN %s ON %s.tx_id = %s.tx_id", other, table, other)
}

func joinOnTokenID(table, other string) string {
	return fmt.Sprintf("LEFT JOIN %s ON %s.tx_id = %s.tx_id AND %s.idx = %s.idx", other, table, other, table, other)
}
