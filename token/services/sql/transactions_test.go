/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb/db/dbtest"
)

func TestTransactionsSqlite(t *testing.T) {
	tempDir := t.TempDir()

	for _, c := range dbtest.Cases {
		initSqlite(t, tempDir, c.Name)
		t.Run(c.Name, func(xt *testing.T) {
			defer Transactions.Close()
			c.Fn(xt, Transactions)
		})
	}
}

func TestTransactionsSqliteMemory(t *testing.T) {
	for _, c := range dbtest.Cases {
		initSqliteMemory(t, c.Name)

		t.Run(c.Name, func(xt *testing.T) {
			defer Transactions.Close()
			c.Fn(xt, Transactions)
		})
	}
}

func TestTransactionsPostgres(t *testing.T) {
	terminate, pgConnStr := startPostgresContainer(t)
	defer terminate()

	for _, c := range dbtest.Cases {
		initPostgres(t, pgConnStr, c.Name)
		t.Run(c.Name, func(xt *testing.T) {
			defer Transactions.Close()
			c.Fn(xt, Transactions)
		})
	}
}
