/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/test-go/testify/assert"
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
	{"Duplicate", TDuplicate},
}

func TDuplicate(t *testing.T, db *WalletDB) {
	id := view.Identity([]byte{254, 0, 155, 1})

	err := db.StoreIdentity(id, "duplicate", "meta")
	assert.NoError(t, err)

	var meta string
	err = db.LoadMeta(id, &meta)
	assert.NoError(t, err)
	assert.Equal(t, "meta", meta)

	err = db.StoreIdentity(id, "duplicate", "")
	assert.NoError(t, err)

	err = db.LoadMeta(id, &meta)
	assert.NoError(t, err)
	assert.Equal(t, "meta", meta)
}
