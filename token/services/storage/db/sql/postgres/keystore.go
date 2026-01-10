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

type KeystoreStore = common3.KeystoreStore

func NewKeystoreStore(dbs *common2.RWDB, tableNames common3.TableNames) (*KeystoreStore, error) {
	return common3.NewKeystoreStore(dbs.ReadDB, dbs.WriteDB, tableNames, postgres.NewConditionInterpreter(), &postgres.ErrorMapper{})
}
