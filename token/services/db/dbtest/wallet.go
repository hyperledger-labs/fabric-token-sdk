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
	t.Helper()
	for _, c := range walletCases {
		driver := cfgProvider(c.Name)
		db, err := driver.NewWallet("", c.Name)
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
	Fn   func(*testing.T, driver.WalletStore)
}{
	{"TDuplicate", TDuplicate},
	{"TWalletIdentities", TWalletIdentities},
}

func TDuplicate(t *testing.T, db driver.WalletStore) {
	t.Helper()
	ctx := t.Context()
	id := []byte{254, 0, 155, 1}

	err := db.StoreIdentity(ctx, id, "eID", "duplicate", 0, []byte("meta"))
	assert.NoError(t, err)

	meta, err := db.LoadMeta(ctx, id, "duplicate", 0)
	assert.NoError(t, err)
	assert.Equal(t, "meta", string(meta))

	err = db.StoreIdentity(ctx, id, "eID", "duplicate", 0, nil)
	assert.NoError(t, err)

	meta, err = db.LoadMeta(ctx, id, "duplicate", 0)
	assert.NoError(t, err)
	assert.Equal(t, "meta", string(meta))
}

func TWalletIdentities(t *testing.T, db driver.WalletStore) {
	t.Helper()
	ctx := t.Context()
	assert.NoError(t, db.StoreIdentity(ctx, []byte("alice"), "eID", "alice_wallet", 0, nil))
	assert.NoError(t, db.StoreIdentity(ctx, []byte("alice"), "eID", "alice_wallet", 1, nil))
	assert.NoError(t, db.StoreIdentity(ctx, []byte("bob"), "eID", "bob_wallet", 0, nil))
	assert.NoError(t, db.StoreIdentity(ctx, []byte("alice"), "eID", "alice_wallet", 0, nil))

	ids, err := db.GetWalletIDs(ctx, 0)
	assert.NoError(t, err)
	assert.Equal(t, []driver.WalletID{"alice_wallet", "bob_wallet"}, ids)

	ids, err = db.GetWalletIDs(ctx, 1)
	assert.NoError(t, err)
	assert.Equal(t, []driver.WalletID{"alice_wallet"}, ids)
}
