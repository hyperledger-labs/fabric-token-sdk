/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package postgres

import (
	"testing"

	sql2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/dbtest"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
)

func initTokenLockDB(dataSourceName, tablePrefix string, maxOpenConns int) (driver.TokenLockDB, driver.TokenTransactionDB, error) {
	d := common.NewSQLDBOpener("", "")
	sqlDB, err := d.OpenSQLDB(sql2.Postgres, dataSourceName, maxOpenConns, false)
	if err != nil {
		return nil, nil, err
	}
	tokenLockDB, err := NewTokenLockDB(sqlDB, common.NewDBOpts{
		DataSource:   dataSourceName,
		TablePrefix:  tablePrefix,
		CreateSchema: true,
	})
	if err != nil {
		return nil, nil, err
	}
	tokenTransactiokDB, err := NewTransactionDB(sqlDB, common.NewDBOpts{
		DataSource:   dataSourceName,
		TablePrefix:  tablePrefix,
		CreateSchema: true,
	})
	if err != nil {
		tokenLockDB.Close()
		return nil, nil, err
	}
	return tokenLockDB, tokenTransactiokDB, err
}

func TestTokenLock(t *testing.T) {
	terminate, pgConnStr := common.StartPostgresContainer(t)
	defer terminate()

	for _, c := range dbtest.TokenLockDBCases {
		tokenLockDB, tokenTransactionDB, err := initTokenLockDB(pgConnStr, c.Name, 10)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(c.Name, func(xt *testing.T) {
			defer tokenLockDB.Close()
			defer tokenTransactionDB.Close()
			c.Fn(xt, tokenLockDB, tokenTransactionDB)
		})
	}
}
