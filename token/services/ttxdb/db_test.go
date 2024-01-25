/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttxdb_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/core/config"
	registry2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/registry"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb/db/sql"
	"github.com/stretchr/testify/assert"
	_ "modernc.org/sqlite"
)

func TestDB(t *testing.T) {
	// create a new config service by loading the config file
	cp, err := config.NewProvider("./testdata/sqlite")
	assert.NoError(t, err)
	registry := registry2.New()
	assert.NoError(t, registry.RegisterService(cp))

	manager := ttxdb.NewManager(registry, "sql")
	db1, err := manager.DBByID("pineapple")
	assert.NoError(t, err)
	db2, err := manager.DBByID("grapes")
	assert.NoError(t, err)

	TEndorserAcks(t, db1, db2)
}

func TEndorserAcks(t *testing.T, db1, db2 *ttxdb.DB) {
	wg := sync.WaitGroup{}
	n := 100
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			assert.NoError(t, db1.AddTransactionEndorsementAck("1", []byte(fmt.Sprintf("alice_%d", i)), []byte(fmt.Sprintf("sigma_%d", i))))
			acks, err := db1.GetTransactionEndorsementAcks("1")
			assert.NoError(t, err)
			assert.True(t, len(acks) != 0)
			assert.NoError(t, db2.AddTransactionEndorsementAck("2", []byte(fmt.Sprintf("bob_%d", i)), []byte(fmt.Sprintf("sigma_%d", i))))
			acks, err = db2.GetTransactionEndorsementAcks("2")
			assert.NoError(t, err)
			assert.True(t, len(acks) != 0)

			wg.Done()
		}(i)
	}
	wg.Wait()

	acks, err := db1.GetTransactionEndorsementAcks("1")
	assert.NoError(t, err)
	assert.Len(t, acks, n)
	for i := 0; i < n; i++ {
		assert.Equal(t, []byte(fmt.Sprintf("sigma_%d", i)), acks[view.Identity(fmt.Sprintf("alice_%d", i)).String()])
	}

	acks, err = db2.GetTransactionEndorsementAcks("2")
	assert.NoError(t, err)
	assert.Len(t, acks, n)
	for i := 0; i < n; i++ {
		assert.Equal(t, []byte(fmt.Sprintf("sigma_%d", i)), acks[view.Identity(fmt.Sprintf("bob_%d", i)).String()])
	}
}
