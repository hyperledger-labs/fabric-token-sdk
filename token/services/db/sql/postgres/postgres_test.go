/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package postgres

import (
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/common/mock"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/multiplexed"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/postgres"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/dbtest"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/test-go/testify/assert"
)

func TestTokens(t *testing.T) {
	terminate, pgConnStr := startContainer(t)
	defer terminate()

	dbtest.TokensTest(t, func(name string) driver.Driver { return NewDriver(postgresCfg(pgConnStr, name)) })
}

func TestTransactions(t *testing.T) {
	terminate, pgConnStr := startContainer(t)
	defer terminate()

	dbtest.TransactionsTest(t, func(name string) driver.Driver { return NewDriver(postgresCfg(pgConnStr, name)) })
}

func TestTokenLocks(t *testing.T) {
	terminate, pgConnStr := startContainer(t)
	defer terminate()

	dbtest.TokenLocksTest(t, func(name string) driver.Driver { return NewDriver(postgresCfg(pgConnStr, name)) })
}

func TestWallet(t *testing.T) {
	terminate, pgConnStr := startContainer(t)
	defer terminate()

	dbtest.WalletTest(t, func(name string) driver.Driver { return NewDriver(postgresCfg(pgConnStr, name)) })
}

func TestIdentity(t *testing.T) {
	terminate, pgConnStr := startContainer(t)
	defer terminate()

	dbtest.IdentityTest(t, func(name string) driver.Driver { return NewDriver(postgresCfg(pgConnStr, name)) })
}

func postgresCfg(pgConnStr string, name string) *mock.ConfigProvider {
	return multiplexed.MockTypeConfig(postgres.Persistence, postgres.Config{
		DataSource:   pgConnStr,
		TablePrefix:  name,
		MaxOpenConns: 10,
	})
}

func startContainer(t *testing.T) (func(), string) {
	t.Helper()
	cfg := postgres.DefaultConfig("test-db")
	terminate, err := postgres.StartPostgresWithFmt([]*postgres.ContainerConfig{cfg})
	assert.NoError(t, err)
	return terminate, cfg.DataSource()
}
