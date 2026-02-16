/*
Copyright IBM Corp. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package sqlite

import (
	"database/sql"
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/sqlite"
	common2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/common"
)

func mockWalletStore(db *sql.DB) *WalletStore {
	store, _ := common2.NewWalletStore(db, db, common2.TableNames{
		Wallets: "WALLETS",
	}, sqlite.NewConditionInterpreter())

	return store
}

func TestGetWalletID(t *testing.T) {
	common2.TestGetWalletID(t, mockWalletStore)
}

func TestGetWalletIDs(t *testing.T) {
	common2.TestGetWalletIDs(t, mockWalletStore)
}

func TestLoadMeta(t *testing.T) {
	common2.TestLoadMeta(t, mockWalletStore)
}

func TestIdentityExists(t *testing.T) {
	common2.TestIdentityExists(t, mockWalletStore)
}

func TestStoreIdentity(t *testing.T) {
	common2.TestStoreIdentity(t, mockWalletStore)
}
