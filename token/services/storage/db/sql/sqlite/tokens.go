/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sqlite

import (
	common2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/notifier"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/sqlite"
	common3 "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/common"
)

type TokenStore = common3.TokenStore

func NewTokenStore(dbs *common2.RWDB, tableNames common3.TableNames) (*TokenStore, error) {
	return common3.NewTokenStore(dbs.ReadDB, dbs.WriteDB, tableNames, sqlite.NewConditionInterpreter())
}

type TokenNotifier struct {
	*notifier.Notifier
}

func NewTokenNotifier(*common2.RWDB, common3.TableNames) (*TokenNotifier, error) {
	return &TokenNotifier{Notifier: notifier.NewNotifier()}, nil
}

func (db *TokenNotifier) CreateSchema() error { return nil }
