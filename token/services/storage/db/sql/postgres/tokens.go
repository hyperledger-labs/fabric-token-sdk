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

type (
	TokenStore    = common3.TokenStore
	TokenNotifier = postgres.Notifier
)

func NewTokenStore(dbs *common2.RWDB, tableNames common3.TableNames) (*TokenStore, error) {
	return common3.NewTokenStore(dbs.ReadDB, dbs.WriteDB, tableNames, postgres.NewConditionInterpreter())
}

func NewTokenNotifier(dbs *common2.RWDB, tableNames common3.TableNames, dataSource string) (*TokenNotifier, error) {
	return postgres.NewNotifier(dbs.WriteDB, tableNames.Tokens, dataSource, postgres.AllOperations, *postgres.NewSimplePrimaryKey("tx_id"), *postgres.NewSimplePrimaryKey("idx")), nil

}
