/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package postgres

import (
	"testing"

	sql2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/common"
	common2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
)

func initTokenNDB(dataSourceName, tablePrefix string, maxOpenConns int) (*TokenNDB, error) {
	tdb, err := NewTokenNDB(common.Opts{
		Driver:          sql2.Postgres,
		DataSource:      dataSourceName,
		TablePrefix:     tablePrefix,
		SkipCreateTable: false,
		SkipPragmas:     false,
		MaxOpenConns:    maxOpenConns,
	})
	if err != nil {
		return nil, err
	}
	return tdb.(*TokenNDB), nil
}

func TestTokensPostgres(t *testing.T) {
	terminate, pgConnStr := common2.StartPostgresContainer(t)
	defer terminate()

	for _, c := range common2.TokenNotifierCases {
		db, err := initTokenNDB(pgConnStr, c.Name, 10)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(c.Name, func(xt *testing.T) {
			defer db.Close()
			c.Fn(xt, db)
		})
	}
}
