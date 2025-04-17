/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dbtest

import (
	"testing"
	"time"

	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/test-go/testify/assert"
)

func TokenLocksTest(t *testing.T, cfgProvider cfgProvider) {
	for _, c := range tokenLockDBCases {
		driver, config := cfgProvider(c.Name)
		tokenLockDB, err := driver.NewTokenLock(config, c.Name)
		if err != nil {
			t.Fatal(err)
		}
		tokenTransactionDB, err := driver.NewOwnerTransaction(config, c.Name)
		if err != nil {
			tokenLockDB.Close()
			t.Fatal(err)
		}
		t.Run(c.Name, func(xt *testing.T) {
			defer tokenLockDB.Close()
			defer tokenTransactionDB.Close()
			c.Fn(xt, tokenLockDB, tokenTransactionDB)
		})
	}
}

var tokenLockDBCases = []struct {
	Name string
	Fn   func(*testing.T, driver.TokenLockDB, driver.TokenTransactionDB)
}{
	{"TestFully", TestFully},
}

func TestFully(t *testing.T, tokenLockDB driver.TokenLockDB, tokenTransactionDB driver.TokenTransactionDB) {
	tx, err := tokenTransactionDB.BeginAtomicWrite()
	assert.NoError(t, err)
	assert.NoError(t, tx.AddTokenRequest("apple", []byte("apple_tx_content"), nil, driver2.PPHash("tr")))
	assert.NoError(t, tx.Commit())

	assert.NoError(t, tokenLockDB.Lock(&token.ID{TxId: "apple", Index: 0}, "pineapple"))
	assert.NoError(t, tokenLockDB.Cleanup(1*time.Second))
}
