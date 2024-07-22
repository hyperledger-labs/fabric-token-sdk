/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"fmt"
	"path"
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/dbtest"
)

func initTransactionsDB(driverName, dataSourceName, tablePrefix string, maxOpenConns int) (*TransactionDB, error) {
	d := NewSQLDBOpener("", "")
	sqlDB, err := d.OpenSQLDB(driverName, dataSourceName, maxOpenConns, false)
	if err != nil {
		return nil, err
	}
	transactionDB, err := NewTransactionDB(sqlDB, driverName, tablePrefix, true)
	if err != nil {
		return nil, err
	}
	return transactionDB.(*TransactionDB), err
}

func TestTransactionsSqlite(t *testing.T) {
	tempDir := t.TempDir()
	for _, c := range dbtest.Cases {
		db, err := initTransactionsDB("sqlite", fmt.Sprintf("file:%s?_pragma=busy_timeout(20000)", path.Join(tempDir, "db.sqlite")), c.Name, 10)
		if err != nil {
			t.Fatal(err)
		}

		t.Run(c.Name, func(xt *testing.T) {
			defer db.Close()
			c.Fn(xt, db)
		})
	}
}

func TestTransactionsSqliteMemory(t *testing.T) {
	for _, c := range dbtest.Cases {
		db, err := initTransactionsDB("sqlite", "file:tmp?_pragma=busy_timeout(20000)&mode=memory&cache=shared", c.Name, 10)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(c.Name, func(xt *testing.T) {
			defer db.Close()
			c.Fn(xt, db)
		})
	}
}

func TestTransactionsPostgres(t *testing.T) {
	terminate, pgConnStr := StartPostgresContainer(t)
	defer terminate()

	for _, c := range dbtest.Cases {
		db, err := initTransactionsDB("pgx", pgConnStr, c.Name, 10)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(c.Name, func(xt *testing.T) {
			defer db.Close()
			c.Fn(xt, db)
		})
	}
}
