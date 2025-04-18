/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identitydb_test

import (
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/core/config"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	db2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/multiplexed"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/sqlite"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identitydb"
	"github.com/stretchr/testify/assert"
)

func TestDB(t *testing.T) {
	// create a new config service by loading the config file
	cp, err := config.NewProvider("./testdata/sqlite")
	assert.NoError(t, err)

	dh := db2.NewDriverHolder(cp, multiplexed.Driver{sqlite.NewNamedDriver()})
	manager := identitydb.NewManager(dh)
	_, err = manager.IdentityDBByTMSId(token2.TMSID{Network: "pineapple"})
	assert.NoError(t, err)
	_, err = manager.WalletDBByTMSId(token2.TMSID{Network: "grapes"})
	assert.NoError(t, err)
}
