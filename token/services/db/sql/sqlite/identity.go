/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sqlite

import (
	common2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/sqlite"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
)

type IdentityStore = common.IdentityStore

func NewIdentityStore(dbs *common2.RWDB, tableNames common.TableNames) (*IdentityStore, error) {
	return common.NewCachedIdentityStore(
		dbs.ReadDB,
		dbs.WriteDB,
		tableNames,
		sqlite.NewConditionInterpreter(),
		&sqlite.ErrorMapper{},
	)
}
