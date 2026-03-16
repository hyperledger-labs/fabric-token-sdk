/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package postgres

import (
	scommon "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/common"
	idriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	sqlcommon "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/common"
)

type IdentityStore = sqlcommon.IdentityStore

type IdentityNotifier struct {
	*Notifier
}

// NewIdentityNotifier returns a new IdentityNotifier for the given RWDB and table names.
func NewIdentityNotifier(dbs *scommon.RWDB, tableNames sqlcommon.TableNames, dataSource string) (*IdentityNotifier, error) {
	return &IdentityNotifier{
		Notifier: NewNotifier(
			dbs.WriteDB,
			tableNames.IdentityConfigurations,
			dataSource,
			AllOperations,
			*NewSimplePrimaryKey("id"),
			*NewSimplePrimaryKey("type"),
			*NewSimplePrimaryKey("url"),
		)}, nil
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
