/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttxdb_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/core/config"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/registry"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/sdk/db"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb/db/sql"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/assert"
	_ "modernc.org/sqlite"
)

func TestDB(t *testing.T) {
	// create a new config service by loading the config file
	cp, err := config.NewProvider("./testdata/sqlite")
	assert.NoError(t, err)
	registry := registry.New()
	assert.NoError(t, registry.RegisterService(cp))

	manager := ttxdb.NewManager(registry, db.NewConfig(cp, "ttxdb.persistence.type"))
	db1, err := manager.DBByTMSId(token.TMSID{Network: "pineapple"})
	assert.NoError(t, err)
	db2, err := manager.DBByTMSId(token.TMSID{Network: "grapes"})
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

type qsMock struct{}

func (qs qsMock) IsMine(id *token2.ID) (bool, error) {
	return true, nil
}

func TestTransactionRecords(t *testing.T) {
	now := time.Now()

	// Transfer
	record := simpleTransfer()
	txr, err := ttxdb.TransactionRecords(&record, now)
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, txr, 1)
	assert.Equal(t, txr[0].ActionType, driver.Transfer)
	assert.Equal(t, txr[0].Amount.Int64(), int64(10))
	assert.Equal(t, txr[0].SenderEID, "alice")
	assert.Equal(t, txr[0].RecipientEID, "bob")
	assert.Equal(t, txr[0].Timestamp, now)
	assert.Equal(t, txr[0].TxID, record.Anchor)

	// Issue
	record = simpleTransfer()
	record.Inputs = token.NewInputStream(qsMock{}, []*token.Input{}, 64)
	txr, err = ttxdb.TransactionRecords(&record, now)
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, txr, 1)
	assert.Equal(t, txr[0].ActionType, driver.Issue)
	assert.Equal(t, txr[0].Amount.Int64(), int64(10))
	assert.Equal(t, txr[0].SenderEID, "")
	assert.Equal(t, txr[0].RecipientEID, "bob")
	assert.Equal(t, txr[0].Timestamp, now)
	assert.Equal(t, txr[0].TxID, record.Anchor)

	// Redeem
	record = redeem()
	txr, err = ttxdb.TransactionRecords(&record, now)
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, txr, 1)
	assert.Equal(t, txr[0].ActionType, driver.Redeem)
	assert.Equal(t, txr[0].Amount.Int64(), int64(10))
	assert.Equal(t, txr[0].SenderEID, "alice")
	assert.Equal(t, txr[0].RecipientEID, "")
	assert.Equal(t, txr[0].Timestamp, now)
	assert.Equal(t, txr[0].TxID, record.Anchor)
}

func simpleTransfer() token.AuditRecord {
	input1 := &token.Input{
		ActionIndex:  0,
		EnrollmentID: "alice",
		Type:         "TOK",
		Quantity:     token2.NewQuantityFromUInt64(10),
	}
	output1 := &token.Output{
		ActionIndex:  0,
		EnrollmentID: "bob",
		Type:         "TOK",
		Quantity:     token2.NewQuantityFromUInt64(10),
	}
	return token.AuditRecord{
		Anchor:  "test",
		Inputs:  token.NewInputStream(qsMock{}, []*token.Input{input1}, 64),
		Outputs: token.NewOutputStream([]*token.Output{output1}, 64),
	}
}

func redeem() token.AuditRecord {
	input1 := &token.Input{
		ActionIndex:  0,
		EnrollmentID: "alice",
		Type:         "TOK",
		Quantity:     token2.NewQuantityFromUInt64(10),
	}
	output1 := &token.Output{
		ActionIndex:  0,
		EnrollmentID: "",
		Type:         "TOK",
		Quantity:     token2.NewQuantityFromUInt64(10),
	}
	return token.AuditRecord{
		Anchor:  "test",
		Inputs:  token.NewInputStream(qsMock{}, []*token.Input{input1}, 64),
		Outputs: token.NewOutputStream([]*token.Output{output1}, 64),
	}
}
