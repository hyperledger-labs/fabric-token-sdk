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
	dbtest2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/dbtest"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/driver"
	"github.com/test-go/testify/require"
)

func TestTokens(t *testing.T) {
	terminate, pgConnStr := startContainer(t)
	defer terminate()

	dbtest2.TokensTest(t, func(name string) driver.Driver { return NewDriver(postgresCfg(pgConnStr, name)) })
}

func TestTransactions(t *testing.T) {
	terminate, pgConnStr := startContainer(t)
	defer terminate()

	dbtest2.TransactionsTest(t, func(name string) driver.Driver { return NewDriver(postgresCfg(pgConnStr, name)) })
}

func TestTokenLocks(t *testing.T) {
	terminate, pgConnStr := startContainer(t)
	defer terminate()

	dbtest2.TokenLocksTest(t, func(name string) driver.Driver { return NewDriver(postgresCfg(pgConnStr, name)) })
}

func TestWallet(t *testing.T) {
	terminate, pgConnStr := startContainer(t)
	defer terminate()

	dbtest2.WalletTest(t, func(name string) driver.Driver { return NewDriver(postgresCfg(pgConnStr, name)) })
}

func TestIdentity(t *testing.T) {
	terminate, pgConnStr := startContainer(t)
	defer terminate()

	dbtest2.IdentityTest(t, func(name string) driver.Driver { return NewDriver(postgresCfg(pgConnStr, name)) })
}

func TestKeyStore(t *testing.T) {
	terminate, pgConnStr := startContainer(t)
	defer terminate()

	dbtest2.KeyStoreTest(t, func(name string) driver.Driver { return NewDriver(postgresCfg(pgConnStr, name)) })
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
	cfg := postgres.DefaultConfig(postgres.WithDBName("test-db"))
	terminate, _, err := postgres.StartPostgres(t.Context(), cfg, nil)
	require.NoError(t, err)

	return terminate, cfg.DataSource()
}
