/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package postgres

import (
	common2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/postgres"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
)

type KeystoreStore = common.KeystoreStore

func NewKeystoreStore(dbs *common2.RWDB, tableNames common.TableNames) (*KeystoreStore, error) {
	return common.NewKeystoreStore(dbs.ReadDB, dbs.WriteDB, tableNames, postgres.NewConditionInterpreter(), &postgres.ErrorMapper{})
}
