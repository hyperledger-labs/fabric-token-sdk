/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dbtest

import (
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/stretchr/testify/assert"
)

func WalletTest(t *testing.T, cfgProvider cfgProvider) {
	for _, c := range walletCases {
		driver, config := cfgProvider(c.Name)
		db, err := driver.NewWallet(config, c.Name)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(c.Name, func(xt *testing.T) {
			c.Fn(xt, db)
		})
		assert.NoError(t, db.Close())
	}
}

var walletCases = []struct {
	Name string
	Fn   func(*testing.T, driver.WalletDB)
}{
	{"TDuplicate", TDuplicate},
	{"TWalletIdentities", TWalletIdentities},
}

func TDuplicate(t *testing.T, db driver.WalletDB) {
	id := []byte{254, 0, 155, 1}

	err := db.StoreIdentity(id, "eID", "duplicate", 0, []byte("meta"))
	assert.NoError(t, err)

	meta, err := db.LoadMeta(id, "duplicate", 0)
	assert.NoError(t, err)
	assert.Equal(t, "meta", string(meta))

	err = db.StoreIdentity(id, "eID", "duplicate", 0, nil)
	assert.NoError(t, err)

	meta, err = db.LoadMeta(id, "duplicate", 0)
	assert.NoError(t, err)
	assert.Equal(t, "meta", string(meta))
}

func TWalletIdentities(t *testing.T, db driver.WalletDB) {
	assert.NoError(t, db.StoreIdentity([]byte("alice"), "eID", "alice_wallet", 0, nil))
	assert.NoError(t, db.StoreIdentity([]byte("alice"), "eID", "alice_wallet", 1, nil))
	assert.NoError(t, db.StoreIdentity([]byte("bob"), "eID", "bob_wallet", 0, nil))
	assert.NoError(t, db.StoreIdentity([]byte("alice"), "eID", "alice_wallet", 0, nil))

	ids, err := db.GetWalletIDs(0)
	assert.NoError(t, err)
	assert.Equal(t, []driver.WalletID{"alice_wallet", "bob_wallet"}, ids)

	ids, err = db.GetWalletIDs(1)
	assert.NoError(t, err)
	assert.Equal(t, []driver.WalletID{"alice_wallet"}, ids)
}
