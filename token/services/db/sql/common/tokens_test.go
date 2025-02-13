/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common_test

import (
	"fmt"
	"path"
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/driver/sql"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/sqlite"
)

//
// func initTokenNDB(driverName common.SQLDriverType, dataSourceName, tablePrefix string, maxOpenConns int) (*TokenNotifier, error) {
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
// }
//
// func initDB[T any](constructor func(db *sql.DB, opts NewDBOpts) (T, error), driverName common.SQLDriverType, dataSourceName, tablePrefix string, maxOpenConns int) (T, error) {
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
// }

func TestTokensSqlite(t *testing.T) {
	tempDir := t.TempDir()
	for _, c := range common.TokensCases {
		db, err := sql.OpenSqlite(common.Opts{
			DataSource:   fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)", path.Join(tempDir, "db.sqlite")),
			TablePrefix:  c.Name,
			MaxOpenConns: 10,
		}, sqlite.NewTokenDB)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(c.Name, func(xt *testing.T) {
			defer db.(*common.TokenDB).Close()
			c.Fn(xt, db.(*common.TokenDB))
		})
	}
	// for _, c := range TokenNotifierCases {
	//	db, err := initTokenNDB(sql2.SQLite, fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)", path.Join(tempDir, "db.sqlite")), c.Name, 10)
	//	if err != nil {
	//		t.Fatal(err)
	//	}
	//	t.Run(c.Name, func(xt *testing.T) {
	//		defer Close(db)
	//		c.Fn(xt, db)
	//	})
	// }
}

func TestTokensSqliteMemory(t *testing.T) {
	for _, c := range common.TokensCases {
		db, err := sql.OpenSqlite(common.Opts{
			DataSource:   "file:tmp?_pragma=busy_timeout(20000)&_pragma=foreign_keys(1)&mode=memory&cache=shared",
			TablePrefix:  c.Name,
			MaxOpenConns: 10,
		}, sqlite.NewTokenDB)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(c.Name, func(xt *testing.T) {
			defer db.(*common.TokenDB).Close()
			c.Fn(xt, db.(*common.TokenDB))
		})
	}
	// for _, c := range TokenNotifierCases {
	//	db, err := initTokenNDB(sql2.SQLite, "file:tmp?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)&mode=memory&cache=shared", c.Name, 10)
	//	if err != nil {
	//		t.Fatal(err)
	//	}
	//	t.Run(c.Name, func(xt *testing.T) {
	//		defer Close(db)
	//		c.Fn(xt, db)
	//	})
	// }
}

func TestTokensPostgres(t *testing.T) {
	terminate, pgConnStr := common.StartPostgresContainer(t)
	defer terminate()

	for _, c := range common.TokensCases {
		db, err := sql.OpenPostgres(common.Opts{
			DataSource:   pgConnStr,
			TablePrefix:  c.Name,
			MaxOpenConns: 10,
		}, sqlite.NewTokenDB)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(c.Name, func(xt *testing.T) {
			defer db.(*common.TokenDB).Close()
			c.Fn(xt, db.(*common.TokenDB))
		})
	}
	// for _, c := range TokenNotifierCases {
	//	db, err := initTokenNDB(sql2.Postgres, pgConnStr, c.Name, 10)
	//	if err != nil {
	//		t.Fatal(err)
	//	}
	//	t.Run(c.Name, func(xt *testing.T) {
	//		defer Close(db)
	//		c.Fn(xt, db)
	//	})
	// }
}
