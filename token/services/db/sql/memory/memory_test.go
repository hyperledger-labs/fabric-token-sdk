/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package memory

import (
	"testing"

	memory "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/memory"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/multiplexed"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/sqlite"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/dbtest"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
)

func TestTokens(t *testing.T) {
	dbtest.TokensTest(t, func(name string) (driver.Driver, driver.Config) {
		return &Driver{}, memoryCfg(name)
	})
}

func TestTransactions(t *testing.T) {
	dbtest.TransactionsTest(t, func(name string) (driver.Driver, driver.Config) {
		return &Driver{}, memoryCfg(name)
	})
}

func TestTokenLocks(t *testing.T) {
	dbtest.TokenLocksTest(t, func(name string) (driver.Driver, driver.Config) {
		return &Driver{}, memoryCfg(name)
	})
}

func TestIdentity(t *testing.T) {
	dbtest.IdentityTest(t, func(name string) (driver.Driver, driver.Config) {
		return &Driver{}, memoryCfg(name)
	})
}

func TestWallet(t *testing.T) {
	dbtest.WalletTest(t, func(name string) (driver.Driver, driver.Config) {
		return &Driver{}, memoryCfg(name)
	})
}

func memoryCfg(name string) *mock.ConfigProvider {
	return multiplexed.MockTypeConfig(memory.Persistence, sqlite.Config{
		DataSource:   "file:tmp?_pragma=busy_timeout(20000)&mode=memory&cache=shared",
		TablePrefix:  name,
		MaxOpenConns: 10,
	})
}
