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

func initTransactionsDB(driverName, dataSourceName string, maxOpenConns int) (*TransactionDB, error) {
	d := NewSQLDBOpener("", "")
	sqlDB, err := d.OpenSQLDB(driverName, dataSourceName, maxOpenConns, false)
	if err != nil {
		return nil, err
	}
	transactionDB, err := NewTransactionDB(sqlDB, true)
	if err != nil {
		return nil, err
	}
	return transactionDB.(*TransactionDB), err
}

func TestTransactionsSqlite(t *testing.T) {
	tempDir := t.TempDir()
	for _, c := range dbtest.Cases {
		db, err := initTransactionsDB("sqlite", fmt.Sprintf("file:%s?_pragma=busy_timeout(20000)", path.Join(tempDir, fmt.Sprintf(c.Name, "db.sqlite"))), 10)
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
		db, err := initTransactionsDB("sqlite", fmt.Sprintf("file:%s?_pragma=busy_timeout(20000)&mode=memory&cache=shared", c.Name), 10)
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
	for _, c := range dbtest.Cases {
		terminate, pgConnStr := StartPostgresContainer(t)
		db, err := initTransactionsDB("pgx", pgConnStr, 10)
		if err != nil {
			terminate()
			t.Fatal(err)
		}
		t.Run(c.Name, func(xt *testing.T) {
			defer db.Close()
			c.Fn(xt, db)
		})
		terminate()
	}
}
