/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tokendb_test

import (
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/core/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/driver/sql"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokendb"
	"github.com/stretchr/testify/assert"
	_ "modernc.org/sqlite"
)

func TestDB(t *testing.T) {
	// create a new config service by loading the config file
	cp, err := config.NewProvider("./testdata/sqlite")
	assert.NoError(t, err)
	dh := db.NewDriverHolder(cp, sql.NewDriver())
	manager := tokendb.NewManager(dh, "tokendb.persistence")
	_, err = manager.DBByTMSId(token.TMSID{Network: "pineapple"})
	assert.NoError(t, err)
	_, err = manager.DBByTMSId(token.TMSID{Network: "grapes"})
	assert.NoError(t, err)
}
