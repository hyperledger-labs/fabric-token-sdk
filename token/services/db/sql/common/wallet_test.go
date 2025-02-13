/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common_test

import (
	"fmt"
	"path"
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/driver/sql"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/sqlite"
	"github.com/stretchr/testify/assert"
)

func TestWalletSqlite(t *testing.T) {
	for _, c := range WalletCases {
		db, err := sql.OpenSqlite(common.Opts{
			DataSource:   fmt.Sprintf("file:%s?_pragma=busy_timeout(20000)", path.Join(t.TempDir(), "db.sqlite")),
			TablePrefix:  c.Name,
			MaxOpenConns: 10,
		}, sqlite.NewWalletDB)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(c.Name, func(xt *testing.T) {
			c.Fn(xt, db)
		})
	}
}

func TestWalletSqliteMemory(t *testing.T) {
	for _, c := range WalletCases {
		db, err := sql.OpenSqlite(common.Opts{
			DataSource:   "file:tmp?_pragma=busy_timeout(20000)&mode=memory&cache=shared",
			TablePrefix:  c.Name,
			MaxOpenConns: 10,
		}, sqlite.NewWalletDB)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(c.Name, func(xt *testing.T) {
			c.Fn(xt, db)
		})
	}
}

func TestWalletPostgres(t *testing.T) {
	terminate, pgConnStr := common.StartPostgresContainer(t)
	defer terminate()

	for _, c := range WalletCases {
		db, err := sql.OpenPostgres(common.Opts{
			DataSource:   pgConnStr,
			TablePrefix:  c.Name,
			MaxOpenConns: 10,
		}, sqlite.NewWalletDB)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(c.Name, func(xt *testing.T) {
			c.Fn(xt, db)
		})
	}
}

var WalletCases = []struct {
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
