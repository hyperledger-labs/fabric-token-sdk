/*
Copyright IBM Corp. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package postgres

import (
	"database/sql"
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/sqlite"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
)

func mockWalletStore(db *sql.DB) *WalletStore {
	store, _ := common.NewWalletStore(db, db, common.TableNames{
		Wallets: "WALLETS",
	}, sqlite.NewConditionInterpreter())
	return store
}

func TestGetWalletID(t *testing.T) {
	common.TestGetWalletID(t, mockWalletStore)
}

func TestGetWalletIDs(t *testing.T) {
	common.TestGetWalletIDs(t, mockWalletStore)
}

func TestLoadMeta(t *testing.T) {
	common.TestLoadMeta(t, mockWalletStore)
}

func TestIdentityExists(t *testing.T) {
	common.TestIdentityExists(t, mockWalletStore)
}

func TestStoreIdentity(t *testing.T) {
	common.TestStoreIdentity(t, mockWalletStore)
}
