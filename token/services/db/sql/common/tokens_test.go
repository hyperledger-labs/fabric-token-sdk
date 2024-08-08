/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"database/sql"
	"fmt"
	"path"
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils"
	sql2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/common"
	sql3 "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql"
)

func initTokenDB(driverName common.SQLDriverType, dataSourceName, tablePrefix string, maxOpenConns int) (*TokenDB, error) {
	d := sql3.NewSQLDBOpener("", "")
	sqlDB, err := d.OpenSQLDB(driverName, dataSourceName, maxOpenConns, false)
	if err != nil {
		return nil, err
	}
	tokenDB, err := NewTokenDB(sqlDB, sql3.NewDBOpts{
		DataSource:   dataSourceName,
		TablePrefix:  tablePrefix,
		CreateSchema: true,
	})
	if err != nil {
		return nil, err
	}
	return tokenDB.(*TokenDB), err
}

func initTokenNDB(driverName common.SQLDriverType, dataSourceName, tablePrefix string, maxOpenConns int) (*TokenNDB, error) {
	d := sql3.NewSQLDBOpener("", "")
	sqlDB, err := d.OpenSQLDB(driverName, dataSourceName, maxOpenConns, false)
	if err != nil {
		return nil, err
	}
	tokenDB, err := NewTokenNDB(sqlDB, sql3.NewDBOpts{
		DataSource:   dataSourceName,
		TablePrefix:  tablePrefix,
		CreateSchema: true,
	})
	if err != nil {
		return nil, err
	}
	return tokenDB.(*TokenNDB), err
}

func initDB[T any](constructor func(db *sql.DB, opts sql3.NewDBOpts) (T, error), driverName common.SQLDriverType, dataSourceName, tablePrefix string, maxOpenConns int) (T, error) {
	d := sql3.NewSQLDBOpener("", "")
	sqlDB, err := d.OpenSQLDB(driverName, dataSourceName, maxOpenConns, false)
	if err != nil {
		return utils.Zero[T](), err
	}
	tokenDB, err := constructor(sqlDB, sql3.NewDBOpts{
		DataSource:   dataSourceName,
		TablePrefix:  tablePrefix,
		CreateSchema: true,
	})
	if err != nil {
		return utils.Zero[T](), err
	}
	return tokenDB, err
}

func TestTokensSqlite(t *testing.T) {
	tempDir := t.TempDir()
	for _, c := range TokensCases {
		db, err := initTokenDB(sql2.SQLite, fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)", path.Join(tempDir, "db.sqlite")), c.Name, 10)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(c.Name, func(xt *testing.T) {
			defer db.Close()
			c.Fn(xt, db)
		})
	}
	for _, c := range TokenNotifierCases {
		db, err := initTokenNDB(sql2.SQLite, fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)", path.Join(tempDir, "db.sqlite")), c.Name, 10)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(c.Name, func(xt *testing.T) {
			defer db.Close()
			c.Fn(xt, db)
		})
	}
}

func TestTokensSqliteMemory(t *testing.T) {
	for _, c := range TokensCases {
		db, err := initTokenDB(sql2.SQLite, "file:tmp?_pragma=busy_timeout(20000)&_pragma=foreign_keys(1)&mode=memory&cache=shared", c.Name, 10)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(c.Name, func(xt *testing.T) {
			defer db.Close()
			c.Fn(xt, db)
		})
	}
	for _, c := range TokenNotifierCases {
		db, err := initTokenNDB(sql2.SQLite, "file:tmp?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)&mode=memory&cache=shared", c.Name, 10)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(c.Name, func(xt *testing.T) {
			defer db.Close()
			c.Fn(xt, db)
		})
	}
}

func TestTokensPostgres(t *testing.T) {
	terminate, pgConnStr := StartPostgresContainer(t)
	defer terminate()

	for _, c := range TokensCases {
		db, err := initTokenDB(sql2.Postgres, pgConnStr, c.Name, 10)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(c.Name, func(xt *testing.T) {
			defer db.Close()
			c.Fn(xt, db)
		})
	}
	for _, c := range TokenNotifierCases {
		db, err := initTokenNDB(sql2.Postgres, pgConnStr, c.Name, 10)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(c.Name, func(xt *testing.T) {
			defer db.Close()
			c.Fn(xt, db)
		})
	}
}
