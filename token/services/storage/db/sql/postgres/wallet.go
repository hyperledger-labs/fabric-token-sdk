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

// WalletStore wraps common.WalletStore to add advisory lock to schema creation
type WalletStore struct {
	*common3.WalletStore
	lockID int64
}

// GetSchema overrides the base GetSchema to prefix with advisory lock
func (s *WalletStore) GetSchema() string {
	baseSchema := s.WalletStore.GetSchema()

	return prefixSchemaWithLock(baseSchema, s.lockID)
}

// NewWalletStore returns a new WalletStore for the given RWDB and table names.
func NewWalletStore(dbs *common2.RWDB, tableNames common3.TableNames) (*WalletStore, error) {
	baseStore, err := common3.NewWalletStore(dbs.ReadDB, dbs.WriteDB, tableNames, postgres.NewConditionInterpreter())
	if err != nil {
		return nil, err
	}

	return &WalletStore{
		WalletStore: baseStore,
		lockID:      createTableLockID("wallet"),
	}, nil
}
