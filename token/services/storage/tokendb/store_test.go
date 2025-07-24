/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tokendb_test

import (
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/config"
	sqlite2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/sqlite"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/multiplexed"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/sqlite"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/tokendb"
	"github.com/stretchr/testify/assert"
	_ "modernc.org/sqlite"
)

func TestDB(t *testing.T) {
	// create a new config service by loading the config file
	cp, err := config.NewProvider("./testdata/sqlite")
	assert.NoError(t, err)
	manager := tokendb.NewStoreServiceManager(cp, multiplexed.NewDriver(cp, sqlite.NewNamedDriver(cp, sqlite2.NewDbProvider())))
	_, err = manager.StoreServiceByTMSId(token.TMSID{Network: "pineapple"})
	assert.NoError(t, err)
	_, err = manager.StoreServiceByTMSId(token.TMSID{Network: "grapes"})
	assert.NoError(t, err)
}
