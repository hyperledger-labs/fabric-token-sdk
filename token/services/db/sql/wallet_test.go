/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/stretchr/testify/assert"
)

func TestWalletSqlite(t *testing.T) {
	tempDir := t.TempDir()

	for _, c := range WalletCases {
		initSqlite(t, tempDir, c.Name)
		t.Run(c.Name, func(xt *testing.T) {
			defer Transactions.Close() // TODO
			c.Fn(xt, Wallet)
		})
	}
}

func TestWalletSqliteMemory(t *testing.T) {
	for _, c := range WalletCases {
		initSqliteMemory(t, c.Name)
		t.Run(c.Name, func(xt *testing.T) {
			defer Transactions.Close()
			c.Fn(xt, Wallet)
		})
	}
}

func TestWalletPostgres(t *testing.T) {
	terminate, pgConnStr := startPostgresContainer(t)
	defer terminate()

	for _, c := range WalletCases {
		initPostgres(t, pgConnStr, c.Name)
		t.Run(c.Name, func(xt *testing.T) {
			defer Transactions.Close()
			c.Fn(xt, Wallet)
		})
	}
}

var WalletCases = []struct {
	Name string
	Fn   func(*testing.T, *WalletDB)
}{
	{"TDuplicate", TDuplicate},
	{"TWalletIdentities", TWalletIdentities},
}

func TDuplicate(t *testing.T, db *WalletDB) {
	id := []byte{254, 0, 155, 1}

	err := db.StoreIdentity(id, "duplicate", 0, []byte("meta"))
	assert.NoError(t, err)

	meta, err := db.LoadMeta(id, "duplicate", 0)
	assert.NoError(t, err)
	assert.Equal(t, "meta", string(meta))

	err = db.StoreIdentity(id, "duplicate", 0, nil)
	assert.NoError(t, err)

	meta, err = db.LoadMeta(id, "duplicate", 0)
	assert.NoError(t, err)
	assert.Equal(t, "meta", string(meta))
}

func TWalletIdentities(t *testing.T, db *WalletDB) {
	assert.NoError(t, db.StoreIdentity([]byte("alice"), "alice_wallet", 0, nil))
	assert.NoError(t, db.StoreIdentity([]byte("alice"), "alice_wallet", 1, nil))
	assert.NoError(t, db.StoreIdentity([]byte("bob"), "bob_wallet", 0, nil))
	assert.NoError(t, db.StoreIdentity([]byte("alice"), "alice_wallet", 0, nil))

	ids, err := db.GetWalletIDs(0)
	assert.NoError(t, err)
	assert.Equal(t, []driver.WalletID{"alice_wallet", "bob_wallet"}, ids)

	ids, err = db.GetWalletIDs(1)
	assert.NoError(t, err)
	assert.Equal(t, []driver.WalletID{"alice_wallet"}, ids)
}
