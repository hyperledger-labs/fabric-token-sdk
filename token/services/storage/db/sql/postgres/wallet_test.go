/*
Copyright IBM Corp. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package postgres

import (
	"database/sql"
	"testing"

	sq "github.com/Masterminds/squirrel"
	common2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/common"
)

func mockWalletStore(db *sql.DB, pf sq.PlaceholderFormat) *common2.WalletStore {
	store, _ := common2.NewWalletStore(db, db, common2.TableNames{
		Wallets: "WALLETS",
	}, pf)

	return store
}

func TestGetWalletID(t *testing.T) {
	common2.TestGetWalletID(t, mockWalletStore, sq.Dollar)
}

func TestGetWalletIDs(t *testing.T) {
	common2.TestGetWalletIDs(t, mockWalletStore, sq.Dollar)
}

func TestLoadMeta(t *testing.T) {
	common2.TestLoadMeta(t, mockWalletStore, sq.Dollar)
}

func TestIdentityExists(t *testing.T) {
	common2.TestIdentityExists(t, mockWalletStore, sq.Dollar)
}

func TestStoreIdentity(t *testing.T) {
	common2.TestStoreIdentity(t, mockWalletStore, sq.Dollar)
}

func TestStoreIdentityIdempotent(t *testing.T) {
	common2.TestStoreIdentityIdempotent(t, mockWalletStore)
}
