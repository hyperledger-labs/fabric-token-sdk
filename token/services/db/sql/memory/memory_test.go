/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package memory

import (
	"context"
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/dbtest"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
)

func TestTokens(t *testing.T) {
	dbtest.TokensTest(context.Background(), t, func(string) driver.Driver { return NewDriver() })
}

func TestTransactions(t *testing.T) {
	dbtest.TransactionsTest(context.Background(), t, func(string) driver.Driver { return NewDriver() })
}

func TestTokenLocks(t *testing.T) {
	dbtest.TokenLocksTest(context.Background(), t, func(string) driver.Driver { return NewDriver() })
}

func TestIdentity(t *testing.T) {
	dbtest.IdentityTest(context.Background(), t, func(string) driver.Driver { return NewDriver() })
}

func TestWallet(t *testing.T) {
	dbtest.WalletTest(context.Background(), t, func(string) driver.Driver { return NewDriver() })
}
