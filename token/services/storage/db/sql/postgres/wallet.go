/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package postgres

import (
	common2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/postgres"
	common3 "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/common"
)

// WalletStore is an alias for common.WalletStore.
type WalletStore = common3.WalletStore

// NewWalletStore returns a new WalletStore for the given RWDB and table names.
func NewWalletStore(dbs *common2.RWDB, tableNames common3.TableNames) (*WalletStore, error) {
	return common3.NewWalletStore(dbs.ReadDB, dbs.WriteDB, tableNames, postgres.NewConditionInterpreter())
}
