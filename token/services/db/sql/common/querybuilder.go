/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
)

func movementConditionsSql(params driver.QueryMovementsParams) string {
	sb := strings.Builder{}

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

	return sb.String()
}

func joinOnTxID(table, other string) string {
	return fmt.Sprintf("LEFT JOIN %s ON %s.tx_id = %s.tx_id", other, table, other)
}

func joinOnTokenID(table, other string) string {
	return fmt.Sprintf("LEFT JOIN %s ON %s.tx_id = %s.tx_id AND %s.idx = %s.idx", other, table, other, table, other)
}
