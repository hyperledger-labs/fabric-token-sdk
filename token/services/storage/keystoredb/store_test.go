/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package keystoredb_test

import (
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/config"
	sqlite2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/sql/sqlite"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/tms"
	config2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/multiplexed"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/sqlite"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/keystoredb"
	"github.com/test-go/testify/require"
)

func TestDB(t *testing.T) {
	// create a new config service by loading the config file
	cp, err := config.NewProvider("./testdata/sqlite")
	require.NoError(t, err)

	manager := keystoredb.NewStoreServiceManager(
		tms.NewConfigServiceWrapper(config2.NewService(cp)),
		multiplexed.NewDriver(cp, sqlite.NewNamedDriver(cp, sqlite2.NewDbProvider())),
	)
	_, err = manager.StoreServiceByTMSId(token2.TMSID{Network: "pineapple", Namespace: "ns"})
	require.NoError(t, err)
}
