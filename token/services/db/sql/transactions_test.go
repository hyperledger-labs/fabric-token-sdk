/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"database/sql"
	"fmt"
	"path"
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb/db/dbtest"
	"github.com/pkg/errors"
)

func initTransactionsDB(driverName, dataSourceName, tablePrefix string, maxOpenConns int) (*TransactionDB, error) {
	db, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open db [%s]", driverName)
	}
	db.SetMaxOpenConns(maxOpenConns)

	if err = db.Ping(); err != nil {
		return nil, errors.Wrapf(err, "failed to ping db [%s]", driverName)
	}
	logger.Infof("connected to [%s:%s] database", driverName, tablePrefix)

	tables, err := getTableNames(tablePrefix)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get table names")
	}
	transactions := newTransactionDB(db, transactionTables{
		Movements:             tables.Movements,
		Transactions:          tables.Transactions,
		Requests:              tables.Requests,
		Validations:           tables.Validations,
		TransactionEndorseAck: tables.TransactionEndorseAck,
	})
	if err = initSchema(db, transactions.GetSchema()); err != nil {
		return transactions, err
	}
	return transactions, nil
}

func TestTransactionsSqlite(t *testing.T) {
	tempDir := t.TempDir()
	for _, c := range dbtest.Cases {
		db, err := initTransactionsDB("sqlite", fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)", path.Join(tempDir, "db.sqlite")), c.Name, 10)
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
		db, err := initTransactionsDB("sqlite", "file:tmp?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&mode=memory&cache=shared", c.Name, 10)
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
	terminate, pgConnStr := startPostgresContainer(t)
	defer terminate()

	for _, c := range dbtest.Cases {
		db, err := initTransactionsDB("postgres", pgConnStr, c.Name, 10)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(c.Name, func(xt *testing.T) {
			defer db.Close()
			c.Fn(xt, db)
		})
	}
}
