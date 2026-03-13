/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sqlite

import (
	common2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/notifier"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/sqlite"
	idriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	common3 "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/common"
)

type IdentityStore = common3.IdentityStore

func NewIdentityStore(dbs *common2.RWDB, tableNames common3.TableNames) (*IdentityStore, error) {
	return common3.NewCachedIdentityStore(
		dbs.ReadDB,
		dbs.WriteDB,
		tableNames,
		sqlite.NewConditionInterpreter(),
		&sqlite.ErrorMapper{},
	)
}

type IdentityNotifier struct {
	*notifier.Notifier
}

func NewIdentityNotifier(*common2.RWDB, common3.TableNames) (*IdentityNotifier, error) {
	return &IdentityNotifier{Notifier: notifier.NewNotifier()}, nil
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

func (db *IdentityNotifier) CreateSchema() error { return nil }
