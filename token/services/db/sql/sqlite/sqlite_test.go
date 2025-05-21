/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sqlite

import (
	"context"
	"fmt"
	"path"
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/multiplexed"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/sqlite"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/dbtest"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
)

func TestTokens(t *testing.T) {
	dbtest.TokensTest(context.Background(), t, func(name string) driver.Driver { return NewDriver(sqliteCfg(t.TempDir(), name)) })
}

func TestTransactions(t *testing.T) {
	dbtest.TransactionsTest(context.Background(), t, func(name string) driver.Driver { return NewDriver(sqliteCfg(t.TempDir(), name)) })
}

func TestTokenLocks(t *testing.T) {
	dbtest.TokenLocksTest(context.Background(), t, func(name string) driver.Driver { return NewDriver(sqliteCfg(t.TempDir(), name)) })
}

func TestIdentity(t *testing.T) {
	dbtest.IdentityTest(context.Background(), t, func(name string) driver.Driver { return NewDriver(sqliteCfg(t.TempDir(), name)) })
}

func TestWallet(t *testing.T) {
	dbtest.WalletTest(context.Background(), t, func(name string) driver.Driver { return NewDriver(sqliteCfg(t.TempDir(), name)) })
}

func sqliteCfg(tempDir string, name string) *mock.ConfigProvider {
	return multiplexed.MockTypeConfig(sqlite.Persistence, sqlite.Config{
		DataSource:   fmt.Sprintf("file:%s?_pragma=busy_timeout(20000)", path.Join(tempDir, "db.sqlite")),
		TablePrefix:  name,
		MaxOpenConns: 10,
	})
}
