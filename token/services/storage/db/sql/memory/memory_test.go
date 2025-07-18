/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package memory

import (
	"testing"

	dbtest2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/dbtest"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/driver"
)

func TestTokens(t *testing.T) {
	dbtest2.TokensTest(t, func(string) driver.Driver { return NewDriver() })
}

func TestTransactions(t *testing.T) {
	dbtest2.TransactionsTest(t, func(string) driver.Driver { return NewDriver() })
}

func TestTokenLocks(t *testing.T) {
	dbtest2.TokenLocksTest(t, func(string) driver.Driver { return NewDriver() })
}

func TestIdentity(t *testing.T) {
	dbtest2.IdentityTest(t, func(string) driver.Driver { return NewDriver() })
}

func TestWallet(t *testing.T) {
	dbtest2.WalletTest(t, func(string) driver.Driver { return NewDriver() })
}
