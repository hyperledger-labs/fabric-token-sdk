/*
Copyright IBM Corp. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/
package common

import (
	common2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/query/common"
)

func NewTestInterpreter() common2.CondInterpreter {
	return newTestInterpreter()
}
