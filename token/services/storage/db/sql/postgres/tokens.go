/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package postgres

import (
	"strconv"

	scommon "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/postgres"
	tokensdriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/driver"
	sqlcommon "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/common"
)

// TokenStore is an alias for common.TokenStore.
type TokenStore = sqlcommon.TokenStore

// TokenNotifier handles notifications for tokens.
type TokenNotifier struct {
	*Notifier
}

// NewTokenNotifier returns a new TokenNotifier for the given RWDB and table names.
func NewTokenNotifier(dbs *scommon.RWDB, tableNames sqlcommon.TableNames, dataSource string) (*TokenNotifier, error) {
	return &TokenNotifier{
		Notifier: NewNotifier(
			dbs.WriteDB,
			tableNames.Tokens,
			dataSource,
			AllOperations,
			*NewSimplePrimaryKey("tx_id"),
			*NewSimplePrimaryKey("idx"),
		)}, nil
}

// Subscribe registers a callback function to be called when a token is inserted, updated, or deleted.
func (n *TokenNotifier) Subscribe(callback func(tokensdriver.Operation, tokensdriver.TokenRecordReference)) error {
	return n.Notifier.Subscribe(func(operation tokensdriver.Operation, m map[tokensdriver.ColumnKey]string) {
		idx, err := strconv.ParseUint(m["idx"], 10, 64)
		if err != nil {
			logger.Errorf("failed to parse token index [%s]: %s", m["idx"], err)

			return
		}
		callback(operation, tokensdriver.TokenRecordReference{
			TxID:  m["tx_id"],
			Index: idx,
		})
	})
}

func NewTokenStoreWithNotifier(dbs *scommon.RWDB, tableNames sqlcommon.TableNames, notifier *TokenNotifier) (*TokenStore, error) {
	return sqlcommon.NewTokenStoreWithNotifier(
		dbs.ReadDB,
		dbs.WriteDB,
		tableNames,
		postgres.NewConditionInterpreter(),
		notifier,
	)
}
