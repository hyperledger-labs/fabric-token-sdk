/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path"
	"runtime/debug"
	"testing"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/cache/secondcache"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
	"github.com/test-go/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	_ "modernc.org/sqlite"
)

var Transactions *TransactionDB
var Tokens *TokenDB
var Identity *IdentityDB
var Wallet *WalletDB

func Init(driverName, dataSourceName, tablePrefix, name string, createSchema bool, maxOpenConns int) error {
	logger.Infof("connecting to sql database [%s:%s]", driverName, tablePrefix) // dataSource can contain a password
	if Transactions != nil {
		Transactions.Close()
	}
	if Tokens != nil {
		Tokens.Close()
	}

	tables, err := getTableNames(tablePrefix, name)
	if err != nil {
		return errors.Wrapf(err, "failed to get table names")
	}

	db, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		return errors.Wrapf(err, "failed to open db [%s]", driverName)
	}
	db.SetMaxOpenConns(maxOpenConns)

	if err = db.Ping(); err != nil {
		return errors.Wrapf(err, "failed to ping db [%s]", driverName)
	}
	logger.Infof("connected to [%s:%s] database", driverName, tablePrefix)

	Transactions = newTransactionDB(db, transactionTables{
		Movements:             tables.Movements,
		Transactions:          tables.Transactions,
		Requests:              tables.Requests,
		Validations:           tables.Validations,
		TransactionEndorseAck: tables.TransactionEndorseAck,
	})
	Tokens = newTokenDB(db, tokenTables{
		Tokens:         tables.Tokens,
		Ownership:      tables.Ownership,
		PublicParams:   tables.PublicParams,
		Certifications: tables.Certifications,
	})
	Identity = newIdentityDB(db, identityTables{
		IdentityConfigurations: tables.IdentityConfigurations,
		AuditInfo:              tables.AuditInfo,
		Signers:                tables.Signers,
	}, secondcache.New(1000))
	Wallet = newWalletDB(db, walletTables{Wallets: tables.Wallets})
	if createSchema {
		if err = initSchema(db, Transactions.GetSchema(), Tokens.GetSchema(), Identity.GetSchema(), Wallet.GetSchema()); err != nil {
			return err
		}
	}
	return nil
}

func TestGetTableNames(t *testing.T) {
	const name = "default,mychannel,tokenchaincode"

	names, err := getTableNames("", name)
	assert.NoError(t, err)
	assert.Equal(t, tableNames{
		Movements:              "movements_5193a5",
		Transactions:           "transactions_5193a5",
		Requests:               "requests_5193a5",
		Validations:            "validations_5193a5",
		TransactionEndorseAck:  "tea_5193a5",
		Certifications:         "certifications_5193a5",
		Tokens:                 "tokens_5193a5",
		Ownership:              "ownership_5193a5",
		PublicParams:           "public_params_5193a5",
		Wallets:                "wallet_5193a5",
		IdentityConfigurations: "dentity_5193a5",
		AuditInfo:              "audit_info_5193a5",
		Signers:                "signers_5193a5",
	}, names)

	names, err = getTableNames("valid_prefix", name)
	assert.NoError(t, err)
	assert.Equal(t, "valid_prefix_transactions_5193a5", names.Transactions)

	names, err = getTableNames("Valid_Prefix", name)
	assert.NoError(t, err)
	assert.Equal(t, "Valid_Prefix_transactions_5193a5", names.Transactions)

	names, err = getTableNames("valid", name)
	assert.NoError(t, err)
	assert.Equal(t, "valid_transactions_5193a5", names.Transactions)

	invalid := []string{
		"invalid;",
		"invalid ",
		"in<valid",
		"in\\valid",
		"in\bvalid",
		"invalid\x00",
		"\"invalid\"",
		"in_valid1",
		"Invalid-Prefix",
	}

	for _, inv := range invalid {
		t.Run(fmt.Sprintf("Prefix: %s", inv), func(t *testing.T) {
			names, err := getTableNames(inv, name)
			assert.NotNil(t, err)
			assert.Equal(t, tableNames{}, names)
		})
	}
}

func initSqlite(t *testing.T, tempDir, key string) {
	if err := Init("sqlite", fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)", path.Join(tempDir, "db.sqlite")), "test", key, true, 10); err != nil {
		t.Fatal(err)
	}
}

func initSqliteMemory(t *testing.T, key string) {
	if err := Init("sqlite", "file:tmp?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&mode=memory&cache=shared", "test", key, true, 10); err != nil {
		t.Fatal(err)
	}
}

func initPostgres(t *testing.T, pgConnStr, key string) {
	if err := Init("postgres", pgConnStr, "test", key, true, 10); err != nil {
		t.Fatal(err)
	}
}

// https://testcontainers.com/guides/getting-started-with-testcontainers-for-go/
// Note: Before running tests: docker pull postgres:16.0-alpine
// Test may time out if image is not present on machine.
func startPostgresContainer(t *testing.T) (func(), string) {
	if os.Getenv("TESTCONTAINERS") != "true" {
		t.Skip("set environment variable TESTCONTAINERS to true to include postgres test")
	}
	if testing.Short() {
		t.Skip("skipping postgres test in short mode")
	}

	ctx := context.Background()
	pg, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:16.0-alpine"),
		testcontainers.WithWaitStrategy(
			wait.ForExposedPort().WithStartupTimeout(30*time.Second)),
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("example"),
	)
	if err != nil {
		t.Fatal(err)
	}
	pgConnStr, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}

	return func() {
		if err := pg.Terminate(ctx); err != nil {
			logger.Errorf("failed to terminate [%s][%s]", err, debug.Stack())
		}
	}, pgConnStr
}
