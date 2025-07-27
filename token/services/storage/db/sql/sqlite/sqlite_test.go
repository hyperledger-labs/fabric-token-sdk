/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sqlite

import (
	"fmt"
	"path"
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/common/mock"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/multiplexed"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/sqlite"
	dbtest2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/dbtest"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/driver"
)

func TestTokens(t *testing.T) {
	dbtest2.TokensTest(t, func(name string) driver.Driver { return NewDriver(sqliteCfg(t.TempDir(), name)) })
}

func TestTransactions(t *testing.T) {
	dbtest2.TransactionsTest(t, func(name string) driver.Driver { return NewDriver(sqliteCfg(t.TempDir(), name)) })
}

func TestTokenLocks(t *testing.T) {
	dbtest2.TokenLocksTest(t, func(name string) driver.Driver { return NewDriver(sqliteCfg(t.TempDir(), name)) })
}

func TestIdentity(t *testing.T) {
	dbtest2.IdentityTest(t, func(name string) driver.Driver { return NewDriver(sqliteCfg(t.TempDir(), name)) })
}

func TestWallet(t *testing.T) {
	dbtest2.WalletTest(t, func(name string) driver.Driver { return NewDriver(sqliteCfg(t.TempDir(), name)) })
}

func sqliteCfg(tempDir string, name string) *mock.ConfigProvider {
	return multiplexed.MockTypeConfig(sqlite.Persistence, sqlite.Config{
		DataSource:   fmt.Sprintf("file:%s?_pragma=busy_timeout(20000)", path.Join(tempDir, "db.sqlite")),
		TablePrefix:  name,
		MaxOpenConns: 10,
	})
}
