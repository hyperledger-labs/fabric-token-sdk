/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sqlite

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/notifier"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/sqlite"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
)

type TokenDB = common.TokenDB

func NewTokenDB(opts sqlite.Opts) (*TokenDB, error) {
	dbs, err := sqlite.DbProvider.OpenDB(opts)
	if err != nil {
		return nil, err
	}
	tableNames, err := common.GetTableNames(opts.TablePrefix, opts.TableNameParams...)
	if err != nil {
		return nil, err
	}
	return common.NewTokenDB(dbs.ReadDB, dbs.WriteDB, tableNames, common.NewTokenInterpreter(sqlite.NewInterpreter()))
}

type TokenNotifier struct {
	*notifier.Notifier
}

func NewTokenNotifier(sqlite.Opts) (*TokenNotifier, error) {
	return &TokenNotifier{Notifier: notifier.NewNotifier()}, nil
}

func (db *TokenNotifier) CreateSchema() error { return nil }
