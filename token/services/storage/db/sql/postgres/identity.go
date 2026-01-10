/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package postgres

import (
	common2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/postgres"
	common3 "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/common"
)

type IdentityStore = common3.IdentityStore

func NewIdentityStore(dbs *common2.RWDB, tableNames common3.TableNames) (*IdentityStore, error) {
	return common3.NewCachedIdentityStore(
		dbs.ReadDB,
		dbs.WriteDB,
		tableNames,
		postgres.NewConditionInterpreter(),
		&postgres.ErrorMapper{},
	)
}
