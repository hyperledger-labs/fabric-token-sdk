/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"context"
	"fmt"
	"os"
	"path"
	"testing"
	"time"

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
		db, err = OpenDB("sqlite", path.Join(tempDir, "db.sqlite"), "test", key, true, false)
	case "sqlite_memory":
		db, err = OpenDB("sqlite", fmt.Sprintf("file:%s?mode=memory&cache=shared", key), "test", key, true, false)
	case "postgres":
		db, err = OpenDB("postgres", pgConnStr, "tsdk", key, true, true)
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

	// Tokens
	for _, c := range dbtest.TokensCases {
		db := getDatabase(t, "sqlite", c.Name)
		t.Run(c.Name, func(xt *testing.T) { c.Fn(xt, db) })
		if err := db.Close(); err != nil {
			t.Error(err)
		}
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
	const transactions = "transactions_5193a5"
	const movements = "movements_5193a5"
	const requests = "requests_5193a5"
	const validations = "validations_5193a5"
	const tea = "tea_5193a5"
	const ownership = "ownership_5193a5"
	const tokens = "tokens_5193a5"
	const audit_tokens = "audit_tokens_5193a5"
	const issued_tokens = "issued_tokens_5193a5"
	const public_params = "public_params_5193a5"

	cases := []struct {
		prefix         string
		expectedResult tableNames
		expectErr      bool
	}{
		{"", tableNames{
			Transactions:          transactions,
			Movements:             movements,
			Requests:              requests,
			Validations:           validations,
			TransactionEndorseAck: tea,
			Ownership:             ownership,
			Tokens:                tokens,
			AuditTokens:           audit_tokens,
			IssuedTokens:          issued_tokens,
			PublicParams:          public_params,
		}, false},
		{"valid_prefix", tableNames{
			Transactions:          "valid_prefix_" + transactions,
			Movements:             "valid_prefix_" + movements,
			Requests:              "valid_prefix_" + requests,
			Validations:           "valid_prefix_" + validations,
			TransactionEndorseAck: "valid_prefix_" + tea,
			Ownership:             "valid_prefix_" + ownership,
			Tokens:                "valid_prefix_" + tokens,
			AuditTokens:           "valid_prefix_" + audit_tokens,
			IssuedTokens:          "valid_prefix_" + issued_tokens,
			PublicParams:          "valid_prefix_" + public_params,
		}, false},
		{"Valid_prefix", tableNames{
			Transactions:          "Valid_prefix_" + transactions,
			Movements:             "Valid_prefix_" + movements,
			Requests:              "Valid_prefix_" + requests,
			Validations:           "Valid_prefix_" + validations,
			TransactionEndorseAck: "Valid_prefix_" + tea,
			Ownership:             "Valid_prefix_" + ownership,
			Tokens:                "Valid_prefix_" + tokens,
			AuditTokens:           "Valid_prefix_" + audit_tokens,
			IssuedTokens:          "Valid_prefix_" + issued_tokens,
			PublicParams:          "Valid_prefix_" + public_params,
		}, false},
		{"valid", tableNames{
			Transactions:          "valid_" + transactions,
			Movements:             "valid_" + movements,
			Requests:              "valid_" + requests,
			Validations:           "valid_" + validations,
			TransactionEndorseAck: "valid_" + tea,
			Ownership:             "valid_" + ownership,
			Tokens:                "valid_" + tokens,
			AuditTokens:           "valid_" + audit_tokens,
			IssuedTokens:          "valid_" + issued_tokens,
			PublicParams:          "valid_" + public_params,
		}, false},
		{"invalid;", tableNames{}, true},
		{"invalid ", tableNames{}, true},
		{"in<valid", tableNames{}, true},
		{"in\\valid", tableNames{}, true},
		{"in\bvalid", tableNames{}, true},
		{"invalid\x00", tableNames{}, true},
		{"\"invalid\"", tableNames{}, true},
		{"in_valid1", tableNames{}, true},
		{"Invalid-Prefix", tableNames{}, true},
	}
	const name = "default,mychannel,tokenchaincode"
	for _, c := range cases {
		t.Run(fmt.Sprintf("Prefix: %s", c.prefix), func(t *testing.T) {
			names, err := getTableNames(c.prefix, name)
			if c.expectErr {
				assert.NotNil(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, c.expectedResult, names)
			}
		})
	}
}
