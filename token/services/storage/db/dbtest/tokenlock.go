/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dbtest

import (
	"testing"
	"time"

	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	driver3 "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/require"
)

func TokenLocksTest(t *testing.T, cfgProvider cfgProvider) {
	t.Helper()
	for _, c := range tokenLockDBCases {
		driver := cfgProvider(c.Name)

		// Create token store first to ensure the tokens table exists
		// This is required because token locks now have a foreign key constraint
		// referencing the tokens table
		tokenDB, err := driver.NewToken("", c.Name)
		if err != nil {
			t.Fatal(err)
		}

		tokenLockDB, err := driver.NewTokenLock("", c.Name)
		if err != nil {
			utils.IgnoreError(tokenDB.Close)
			t.Fatal(err)
		}
		tokenTransactionDB, err := driver.NewOwnerTransaction("", c.Name)
		if err != nil {
			utils.IgnoreError(tokenDB.Close)
			utils.IgnoreError(tokenLockDB.Close)
			t.Fatal(err)
		}
		t.Run(c.Name, func(xt *testing.T) {
			defer utils.IgnoreError(tokenDB.Close)
			defer utils.IgnoreError(tokenLockDB.Close)
			defer utils.IgnoreError(tokenTransactionDB.Close)
			c.Fn(xt, tokenLockDB, tokenTransactionDB)
		})
	}
}

var tokenLockDBCases = []struct {
	Name string
	Fn   func(*testing.T, driver3.TokenLockStore, driver3.TokenTransactionStore)
}{
	{"TestFully", TestFully},
}

func TestFully(t *testing.T, tokenLockDB driver3.TokenLockStore, tokenTransactionDB driver3.TokenTransactionStore) {
	ctx := t.Context()

	// First, create a token request in the transaction store
	tx, err := tokenTransactionDB.NewTransactionStoreTransaction()
	require.NoError(t, err)
	require.NoError(t, tx.AddTokenRequest(ctx, "apple", []byte("apple_tx_content"), nil, nil, driver2.PPHash("tr")))
	require.NoError(t, tx.Commit())

	// Lock a token - this will succeed even if the token doesn't exist in the tokens table
	// because we removed the foreign key constraint to avoid schema initialization order issues.
	// In production, the application logic ensures tokens exist before they are locked.
	err = tokenLockDB.Lock(ctx, &token.ID{TxId: "apple", Index: 0}, "pineapple")
	require.NoError(t, err, "Lock should succeed")

	// Unlock the token by transaction ID
	err = tokenLockDB.UnlockByTxID(ctx, "pineapple")
	require.NoError(t, err, "Unlock should succeed")

	// Cleanup should work correctly
	require.NoError(t, tokenLockDB.Cleanup(ctx, 1*time.Second))
}
