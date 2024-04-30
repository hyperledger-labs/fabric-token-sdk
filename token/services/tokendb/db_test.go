/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tokendb_test

import (
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/core/config"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/db"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokendb"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/tokendb/db/sql"
	"github.com/stretchr/testify/assert"
	_ "modernc.org/sqlite"
)

func TestDB(t *testing.T) {
	// create a new config service by loading the config file
	cp, err := config.NewProvider("./testdata/sqlite")
	assert.NoError(t, err)

	manager := tokendb.NewManager(cp, db.NewConfig(cp, "tokendb.persistence.type"))
	_, err = manager.DBByTMSId(token2.TMSID{Network: "pineapple"})
	assert.NoError(t, err)
	_, err = manager.DBByTMSId(token2.TMSID{Network: "grapes"})
	assert.NoError(t, err)
}
