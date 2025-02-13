/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common_test

import (
	"fmt"
	"path"
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/dbtest"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/driver/sql"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/postgres"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/sqlite"
)

func TestTokenLockSqlite(t *testing.T) {
	for _, c := range dbtest.TokenLockDBCases {
		tempDir := t.TempDir()
		tokenLockDB, err := sql.OpenSqlite(common.Opts{
			DataSource:   fmt.Sprintf("file:%s?_pragma=busy_timeout(20000)", path.Join(tempDir, "db.sqlite")),
			TablePrefix:  c.Name,
			MaxOpenConns: 10,
		}, sqlite.NewTokenLockDB)
		if err != nil {
			t.Fatal(err)
		}
		tokenTransactionDB, err := sql.OpenSqlite(common.Opts{
			DataSource:   fmt.Sprintf("file:%s?_pragma=busy_timeout(20000)", path.Join(tempDir, "db.sqlite")),
			TablePrefix:  c.Name,
			MaxOpenConns: 10,
		}, sqlite.NewTransactionDB)
		if err != nil {
			tokenLockDB.Close()
			t.Fatal(err)
		}
		t.Run(c.Name, func(xt *testing.T) {
			defer tokenLockDB.Close()
			defer tokenTransactionDB.Close()
			c.Fn(xt, tokenLockDB, tokenTransactionDB)
		})
	}
}

func TestTokenLockMemory(t *testing.T) {
	for _, c := range dbtest.TokenLockDBCases {
		tokenLockDB, err := sql.OpenSqlite(common.Opts{
			DataSource:   "file:tmp?_pragma=busy_timeout(20000)&mode=memory&cache=shared",
			TablePrefix:  c.Name,
			MaxOpenConns: 10,
		}, sqlite.NewTokenLockDB)
		if err != nil {
			t.Fatal(err)
		}
		tokenTransactionDB, err := sql.OpenSqlite(common.Opts{
			DataSource:   "file:tmp?_pragma=busy_timeout(20000)&mode=memory&cache=shared",
			TablePrefix:  c.Name,
			MaxOpenConns: 10,
		}, sqlite.NewTransactionDB)
		if err != nil {
			tokenLockDB.Close()
			t.Fatal(err)
		}
		t.Run(c.Name, func(xt *testing.T) {
			defer tokenLockDB.Close()
			defer tokenTransactionDB.Close()
			c.Fn(xt, tokenLockDB, tokenTransactionDB)
		})
	}
}

func TestTokenLockPostgres(t *testing.T) {
	terminate, pgConnStr := common.StartPostgresContainer(t)
	defer terminate()

	for _, c := range dbtest.TokenLockDBCases {
		tokenLockDB, err := sql.OpenPostgres(common.Opts{
			DataSource:   pgConnStr,
			TablePrefix:  c.Name,
			MaxOpenConns: 10,
		}, postgres.NewTokenLockDB)
		if err != nil {
			t.Fatal(err)
		}
		tokenTransactionDB, err := sql.OpenPostgres(common.Opts{
			DataSource:   pgConnStr,
			TablePrefix:  c.Name,
			MaxOpenConns: 10,
		}, postgres.NewTransactionDB)
		if err != nil {
			tokenLockDB.Close()
			t.Fatal(err)
		}
		t.Run(c.Name, func(xt *testing.T) {
			defer tokenLockDB.Close()
			defer tokenTransactionDB.Close()
			c.Fn(xt, tokenLockDB, tokenTransactionDB)
		})
	}
}
