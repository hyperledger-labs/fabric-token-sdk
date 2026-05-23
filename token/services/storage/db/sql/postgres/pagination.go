/*
Copyright IBM Corp. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/
package postgres

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/query/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/query/pagination"
)

func NewPaginationInterpreter() common.PagInterpreter {
	return pagination.NewDefaultInterpreter()
}
