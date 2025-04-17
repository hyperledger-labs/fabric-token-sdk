/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package postgres

import (
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/multiplexed"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/postgres"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/sqlite"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/dbtest"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
)

func TestTokens(t *testing.T) {
	terminate, pgConnStr := StartContainer(t)
	defer terminate()

	dbtest.TokensTest(t, func(name string) (driver.Driver, driver.Config) {
		return &Driver{}, postgresCfg(pgConnStr, name)
	})
}

func TestTransactions(t *testing.T) {
	terminate, pgConnStr := StartContainer(t)
	defer terminate()

	dbtest.TransactionsTest(t, func(name string) (driver.Driver, driver.Config) {
		return &Driver{}, postgresCfg(pgConnStr, name)
	})
}

func TestTokenLocks(t *testing.T) {
	terminate, pgConnStr := StartContainer(t)
	defer terminate()

	dbtest.TokenLocksTest(t, func(name string) (driver.Driver, driver.Config) {
		return &Driver{}, postgresCfg(pgConnStr, name)
	})
}

func TestWallet(t *testing.T) {
	terminate, pgConnStr := StartContainer(t)
	defer terminate()

	dbtest.WalletTest(t, func(name string) (driver.Driver, driver.Config) {
		return &Driver{}, postgresCfg(pgConnStr, name)
	})
}

func TestIdentity(t *testing.T) {
	terminate, pgConnStr := StartContainer(t)
	defer terminate()

	dbtest.IdentityTest(t, func(name string) (driver.Driver, driver.Config) {
		return &Driver{}, postgresCfg(pgConnStr, name)
	})
}

func postgresCfg(pgConnStr string, name string) *mock.ConfigProvider {
	return multiplexed.MockTypeConfig(postgres.Persistence, sqlite.Config{
		DataSource:   pgConnStr,
		TablePrefix:  name,
		MaxOpenConns: 10,
	})
}
