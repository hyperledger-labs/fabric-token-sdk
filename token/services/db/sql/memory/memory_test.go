/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package memory

import (
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/dbtest"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
)

func TestTokens(t *testing.T) {
	dbtest.TokensTest(t, func(string) driver.Driver { return NewDriver() })
}

func TestTransactions(t *testing.T) {
	dbtest.TransactionsTest(t, func(string) driver.Driver { return NewDriver() })
}

func TestTokenLocks(t *testing.T) {
	dbtest.TokenLocksTest(t, func(string) driver.Driver { return NewDriver() })
}

func TestIdentity(t *testing.T) {
	dbtest.IdentityTest(t, func(string) driver.Driver { return NewDriver() })
}

func TestKeyStore(t *testing.T) {
	dbtest.KeyStoreTest(t, func(string) driver.Driver { return NewDriver() })
}

func TestWallet(t *testing.T) {
	dbtest.WalletTest(t, func(string) driver.Driver { return NewDriver() })
}
