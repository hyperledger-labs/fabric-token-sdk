/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql_test

import (
	"context"
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb/db/sql"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb/db/dbtest"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb/driver"
	_ "github.com/lib/pq"
	"github.com/test-go/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	_ "modernc.org/sqlite"
)

var pgConnStr string
var tempDir string

func getDatabase(t *testing.T, typ string, key string) (db driver.TokenTransactionDB) {
	var err error
	switch typ {
	case "sqlite":
		db, err = sql.OpenDB("sqlite", path.Join(tempDir, "db.sqlite"), "test", key, true)
	case "sqlite_memory":
		db, err = sql.OpenDB("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", key), "test", key, true)
	case "postgres":
		db, err = sql.OpenDB("postgres", pgConnStr, "tsdk", key, true)
	}
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestSqlite(t *testing.T) {
	var err error
	tempDir, err = os.MkdirTemp("", "sql-token-test")
	if err != nil {
		t.Fatalf("failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	for _, c := range dbtest.Cases {
		db := getDatabase(t, "sqlite", c.Name)
		t.Run(c.Name, func(xt *testing.T) {
			defer db.Close()
			c.Fn(xt, db)
		})
	}
}

func TestMemory(t *testing.T) {
	for _, c := range dbtest.Cases {
		db := getDatabase(t, "sqlite_memory", c.Name)
		t.Run(c.Name, func(xt *testing.T) {
			defer db.Close()
			c.Fn(xt, db)
		})
	}
}

func TestPostgres(t *testing.T) {
	if os.Getenv("TESTCONTAINERS") != "true" {
		t.Skip("set environment variable TESTCONTAINERS to true to include postgres test")
	}
	if testing.Short() {
		t.Skip("skipping postgres test in short mode")
	}
	ctx := context.Background()

	// https://testcontainers.com/guides/getting-started-with-testcontainers-for-go/
	// Note: Before running tests: docker pull postgres:16.0-alpine
	// Test may time out if image is not present on machine.
	pg, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:16.0-alpine"),
		testcontainers.WithWaitStrategy(
			wait.ForExposedPort().WithStartupTimeout(10*time.Second)),
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("example"),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer pg.Terminate(ctx)

	pgConnStr, err = pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range dbtest.Cases {
		db := getDatabase(t, "postgres", c.Name)
		t.Run(c.Name, func(xt *testing.T) {
			defer db.Close()
			c.Fn(xt, db)
		})
	}
}

func TestGetTableNames(t *testing.T) {
	cases := []struct {
		prefix         string
		expectedResult sql.TableNames
		expectErr      bool
	}{
		{"valid_prefix", sql.TableNames{Transactions: "valid_prefix_transactions_5193a5", Movements: "valid_prefix_movements_5193a5", Requests: "valid_prefix_requests_5193a5", Validations: "valid_prefix_validations_5193a5"}, false},
		{"Valid_prefix", sql.TableNames{Transactions: "Valid_prefix_transactions_5193a5", Movements: "Valid_prefix_movements_5193a5", Requests: "Valid_prefix_requests_5193a5", Validations: "Valid_prefix_validations_5193a5"}, false},
		{"valid", sql.TableNames{Transactions: "valid_transactions_5193a5", Movements: "valid_movements_5193a5", Requests: "valid_requests_5193a5", Validations: "valid_validations_5193a5"}, false},
		{"invalid;", sql.TableNames{}, true},
		{"invalid ", sql.TableNames{}, true},
		{"in<valid", sql.TableNames{}, true},
		{"in\\valid", sql.TableNames{}, true},
		{"in\bvalid", sql.TableNames{}, true},
		{"invalid\x00", sql.TableNames{}, true},
		{"\"invalid\"", sql.TableNames{}, true},
		{"in_valid1", sql.TableNames{}, true},
		{"Invalid-Prefix", sql.TableNames{}, true},
	}
	const name = "default,mychannel,tokenchaincode"
	for _, c := range cases {
		t.Run(fmt.Sprintf("Prefix: %s", c.prefix), func(t *testing.T) {
			names, err := sql.GetTableNames(c.prefix, name)
			if c.expectErr {
				assert.NotNil(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, c.expectedResult, names)
			}
		})
	}
}
