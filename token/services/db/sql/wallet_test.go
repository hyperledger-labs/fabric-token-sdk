/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"fmt"
	"path"
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/stretchr/testify/assert"
)

func initWalletDB(driverName, dataSourceName, tablePrefix string, maxOpenConns int) (*WalletDB, error) {
	d := NewSQLDBOpener("", "")
	sqlDB, err := d.OpenSQLDB(driverName, dataSourceName, maxOpenConns, false)
	if err != nil {
		return nil, err
	}
	walletDB, err := NewWalletDB(sqlDB, tablePrefix, true)
	if err != nil {
		return nil, err
	}
	return walletDB.(*WalletDB), nil
}

func TestWalletSqlite(t *testing.T) {
	tempDir := t.TempDir()

	for _, c := range WalletCases {
		db, err := initWalletDB("sqlite", fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)", path.Join(tempDir, "db.sqlite")), c.Name, 10)
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
		db, err := initWalletDB("sqlite", "file:tmp?_pragma=busy_timeout(5000)&mode=memory&cache=shared", c.Name, 10)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(c.Name, func(xt *testing.T) {
			c.Fn(xt, db)
		})
	}
}

func TestWalletPostgres(t *testing.T) {
	terminate, pgConnStr := StartPostgresContainer(t)
	defer terminate()

	for _, c := range WalletCases {
		db, err := initWalletDB("postgres", pgConnStr, c.Name, 10)
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
	Fn   func(*testing.T, *WalletDB)
}{
	{"TDuplicate", TDuplicate},
	{"TWalletIdentities", TWalletIdentities},
}

func TDuplicate(t *testing.T, db *WalletDB) {
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

func TWalletIdentities(t *testing.T, db *WalletDB) {
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
