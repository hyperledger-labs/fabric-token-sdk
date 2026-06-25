/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sqlite

import (
	common2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/common"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/query/pagination"
)

type EndorserStore = common.EndorserStore

// NewEndorserStore creates a new EndorserStore for SQLite
func NewEndorserStore(dbs *common2.RWDB, tables common.TableNames) (*EndorserStore, error) {
	return common.NewEndorserStore(
		dbs.ReadDB,
		dbs.WriteDB,
		tables,
		NewConditionInterpreter(),
		pagination.NewDefaultInterpreter(),
	)
}

var _ driver2.EndorserStore = (*EndorserStore)(nil)
