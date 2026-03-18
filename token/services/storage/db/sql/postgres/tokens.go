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

// TokenStore is an alias for common.TokenStore.
type TokenStore = common3.TokenStore

// TokenNotifier is an alias for Notifier.
type TokenNotifier = Notifier

// NewTokenStore returns a new TokenStore for the given RWDB and table names.
func NewTokenStore(dbs *common2.RWDB, tableNames common3.TableNames) (*TokenStore, error) {
	return common3.NewTokenStore(dbs.ReadDB, dbs.WriteDB, tableNames, postgres.NewConditionInterpreter())
}

// NewTokenNotifier returns a new TokenNotifier for the given RWDB and table names.
func NewTokenNotifier(dbs *common2.RWDB, tableNames common3.TableNames, dataSource string) (*TokenNotifier, error) {
	return NewNotifier(
		dbs.WriteDB,
		tableNames.Tokens,
		dataSource,
		AllOperations,
		*NewSimplePrimaryKey("tx_id"),
		*NewSimplePrimaryKey("idx"),
	), nil
}
