/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identitydb_test

import (
	"testing"

	config2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/config"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/core/config"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/db"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identitydb"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/identitydb/db/sql"
	"github.com/stretchr/testify/assert"
	_ "modernc.org/sqlite"
)

func TestDB(t *testing.T) {
	// create a new config service by loading the config file
	cp, err := config.NewProvider("./testdata/sqlite")
	assert.NoError(t, err)

	manager := identitydb.NewManager(cp, db.NewConfig(config2.NewService(cp), "identitydb.persistence.type"))
	_, err = manager.IdentityDBByTMSId(token2.TMSID{Network: "pineapple"})
	assert.NoError(t, err)
	_, err = manager.WalletDBByTMSId(token2.TMSID{Network: "grapes"})
	assert.NoError(t, err)
}
