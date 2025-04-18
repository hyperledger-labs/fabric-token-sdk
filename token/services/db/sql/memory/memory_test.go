/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package memory

import (
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/common"
	memory "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/memory"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/dbtest"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
)

var (
	config = common.MockConfig(memory.Persistence)
)

func TestTokens(t *testing.T) {
	dbtest.TokensTest(t, func(name string) (driver.Driver, driver.Config) {
		return NewDriver(), config
	})
}

func TestTransactions(t *testing.T) {
	dbtest.TransactionsTest(t, func(name string) (driver.Driver, driver.Config) {
		return NewDriver(), config
	})
}

func TestTokenLocks(t *testing.T) {
	dbtest.TokenLocksTest(t, func(name string) (driver.Driver, driver.Config) {
		return NewDriver(), config
	})
}

func TestIdentity(t *testing.T) {
	dbtest.IdentityTest(t, func(name string) (driver.Driver, driver.Config) {
		return NewDriver(), config
	})
}

func TestWallet(t *testing.T) {
	dbtest.WalletTest(t, func(name string) (driver.Driver, driver.Config) {
		return NewDriver(), config
	})
}
