/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tokendb_test

import (
	"testing"

	config2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/config"
	db2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/core/config"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/db"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokendb"
	tokendb2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/tokendb/db/sql"
	"github.com/stretchr/testify/assert"
	_ "modernc.org/sqlite"
)

func TestDB(t *testing.T) {
	// create a new config service by loading the config file
	cp, err := config.NewProvider("./testdata/sqlite")
	assert.NoError(t, err)
	manager := tokendb.NewHolder([]db2.NamedDriver[driver2.TokenDBDriver]{tokendb2.NewDriver()}).
		NewManager(cp, db.NewConfig(config2.NewService(cp), "tokendb.persistence.type"))
	_, err = manager.DBByTMSId(token2.TMSID{Network: "pineapple"})
	assert.NoError(t, err)
	_, err = manager.DBByTMSId(token2.TMSID{Network: "grapes"})
	assert.NoError(t, err)
}
