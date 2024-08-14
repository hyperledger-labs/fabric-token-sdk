/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"fmt"
	"path"
	"testing"

	sql2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/common"
)

func initTokenDB(driverName common.SQLDriverType, dataSourceName, tablePrefix string, maxOpenConns int) (*TokenDB, error) {
	d := NewSQLDBOpener("", "")
	sqlDB, err := d.OpenSQLDB(driverName, dataSourceName, maxOpenConns, false)
	if err != nil {
		return nil, err
	}
	tokenDB, err := NewTokenDB(sqlDB, NewDBOpts{
		DataSource:   dataSourceName,
		TablePrefix:  tablePrefix,
		CreateSchema: true,
	})
	if err != nil {
		return nil, err
	}
	return tokenDB.(*TokenDB), err
}

//
//func initTokenNDB(driverName common.SQLDriverType, dataSourceName, tablePrefix string, maxOpenConns int) (*TokenNotifier, error) {
//	d := NewSQLDBOpener("", "")
//	sqlDB, err := d.OpenSQLDB(driverName, dataSourceName, maxOpenConns, false)
//	if err != nil {
//		return nil, err
//	}
//	tokenDB, err := NewTokenNotifier(sqlDB, NewDBOpts{
//		DataSource:   dataSourceName,
//		TablePrefix:  tablePrefix,
//		CreateSchema: true,
//	})
//	if err != nil {
//		return nil, err
//	}
//	return tokenDB.(*TokenNotifier), err
//}
//
//func initDB[T any](constructor func(db *sql.DB, opts NewDBOpts) (T, error), driverName common.SQLDriverType, dataSourceName, tablePrefix string, maxOpenConns int) (T, error) {
//	d := NewSQLDBOpener("", "")
//	sqlDB, err := d.OpenSQLDB(driverName, dataSourceName, maxOpenConns, false)
//	if err != nil {
//		return utils.Zero[T](), err
//	}
//	tokenDB, err := constructor(sqlDB, NewDBOpts{
//		DataSource:   dataSourceName,
//		TablePrefix:  tablePrefix,
//		CreateSchema: true,
//	})
//	if err != nil {
//		return utils.Zero[T](), err
//	}
//	return tokenDB, err
//}

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
	//for _, c := range TokenNotifierCases {
	//	db, err := initTokenNDB(sql2.SQLite, fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)", path.Join(tempDir, "db.sqlite")), c.Name, 10)
	//	if err != nil {
	//		t.Fatal(err)
	//	}
	//	t.Run(c.Name, func(xt *testing.T) {
	//		defer db.Close()
	//		c.Fn(xt, db)
	//	})
	//}
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
	//for _, c := range TokenNotifierCases {
	//	db, err := initTokenNDB(sql2.SQLite, "file:tmp?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)&mode=memory&cache=shared", c.Name, 10)
	//	if err != nil {
	//		t.Fatal(err)
	//	}
	//	t.Run(c.Name, func(xt *testing.T) {
	//		defer db.Close()
	//		c.Fn(xt, db)
	//	})
	//}
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
	//for _, c := range TokenNotifierCases {
	//	db, err := initTokenNDB(sql2.Postgres, pgConnStr, c.Name, 10)
	//	if err != nil {
	//		t.Fatal(err)
	//	}
	//	t.Run(c.Name, func(xt *testing.T) {
	//		defer db.Close()
	//		c.Fn(xt, db)
	//	})
	//}
}
