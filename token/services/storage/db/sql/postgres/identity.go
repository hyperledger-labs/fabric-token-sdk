/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package postgres

import (
	common2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/postgres"
	idriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	common3 "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/common"
)

type IdentityStore = common3.IdentityStore

type IdentityNotifier struct {
	*postgres.Notifier
}

func NewIdentityStore(dbs *common2.RWDB, tableNames common3.TableNames) (*IdentityStore, error) {
	return common3.NewCachedIdentityStore(
		dbs.ReadDB,
		dbs.WriteDB,
		tableNames,
		postgres.NewConditionInterpreter(),
		&postgres.ErrorMapper{},
	)
}

// NewIdentityNotifier returns a new IdentityNotifier for the given RWDB and table names.
func NewIdentityNotifier(dbs *common2.RWDB, tableNames common3.TableNames, dataSource string) (*IdentityNotifier, error) {
	return &IdentityNotifier{Notifier: postgres.NewNotifier(dbs.WriteDB, tableNames.IdentityConfigurations, dataSource, postgres.AllOperations, *postgres.NewSimplePrimaryKey("id"), *postgres.NewSimplePrimaryKey("type"), *postgres.NewSimplePrimaryKey("url"))}, nil
}

// Subscribe registers a callback function to be called when an identity configuration is inserted or updated.
func (n *IdentityNotifier) Subscribe(callback func(idriver.Operation, idriver.IdentityConfigurationRecord)) error {
	return n.Notifier.Subscribe(func(operation idriver.Operation, m map[idriver.ColumnKey]string) {
		callback(operation, idriver.IdentityConfigurationRecord{
			ID:   m["id"],
			Type: m["type"],
			URL:  m["url"],
		})
	})
}
