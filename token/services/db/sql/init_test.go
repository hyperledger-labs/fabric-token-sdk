/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/test-go/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	_ "modernc.org/sqlite"
)

func TestGetTableNames(t *testing.T) {
	names, err := getTableNames("")
	assert.NoError(t, err)
	assert.Equal(t, tableNames{
		Movements:              "movements",
		Transactions:           "transactions",
		Requests:               "requests",
		Validations:            "validations",
		TransactionEndorseAck:  "tea",
		Certifications:         "certifications",
		Tokens:                 "tokens",
		Ownership:              "ownership",
		PublicParams:           "public_params",
		Wallets:                "wallet",
		IdentityConfigurations: "id_configs",
		IdentityInfo:           "id_info",
		Signers:                "signers",
	}, names)

	names, err = getTableNames("valid_prefix")
	assert.NoError(t, err)
	assert.Equal(t, "valid_prefix_transactions", names.Transactions)

	names, err = getTableNames("Valid_Prefix")
	assert.NoError(t, err)
	assert.Equal(t, "Valid_Prefix_transactions", names.Transactions)

	names, err = getTableNames("valid")
	assert.NoError(t, err)
	assert.Equal(t, "valid_transactions", names.Transactions)

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
			names, err := getTableNames(inv)
			assert.NotNil(t, err)
			assert.Equal(t, tableNames{}, names)
		})
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
