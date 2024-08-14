/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dbtest

import (
	"testing"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/test-go/testify/assert"
)

// TokenLockDBCases collects test functions that db driver implementations can use for integration tests
var TokenLockDBCases = []struct {
	Name string
	Fn   func(*testing.T, driver.TokenLockDB, driver.TokenTransactionDB)
}{
	{"TestFully", TestFully},
}

func TestFully(t *testing.T, tokenLockDB driver.TokenLockDB, tokenTransactionDB driver.TokenTransactionDB) {
	tx, err := tokenTransactionDB.BeginAtomicWrite()
	assert.NoError(t, err)
	assert.NoError(t, tx.AddTokenRequest("apple", []byte("apple_tx_content"), nil))
	assert.NoError(t, tx.Commit())

	assert.NoError(t, tokenLockDB.Lock(&token.ID{TxId: "apple", Index: 0}, "pineapple"))
	assert.NoError(t, tokenLockDB.Cleanup(1*time.Second))
}
