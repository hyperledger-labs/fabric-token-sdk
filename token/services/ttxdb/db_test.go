/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttxdb_test

import (
	"fmt"
	"math/big"
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

	manager := ttxdb.NewManager(cp, db.NewConfig(cp, "ttxdb.persistence.type"))
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
	input := simpleTransfer()
	recs, err := ttxdb.TransactionRecords(&input, now)
	assert.NoError(t, err)
	assert.Equal(t, []driver.TransactionRecord{
		{
			TxID:         input.Anchor,
			ActionType:   driver.Transfer,
			SenderEID:    "alice",
			RecipientEID: "bob",
			TokenType:    "TOK",
			Amount:       big.NewInt(10),
			Timestamp:    now,
			Status:       driver.Pending,
		},
	}, recs)

	// Transfer with change
	input = transferWithChange()
	recs, err = ttxdb.TransactionRecords(&input, now)
	if err != nil {
		t.Fatal(err)
	}
	assert.NoError(t, err)
	assert.Equal(t, []driver.TransactionRecord{
		{
			TxID:         input.Anchor,
			ActionType:   driver.Transfer,
			SenderEID:    "alice",
			RecipientEID: "bob",
			TokenType:    "TOK",
			Amount:       big.NewInt(10),
			Timestamp:    now,
			Status:       driver.Pending,
		},
		{
			TxID:         input.Anchor,
			ActionType:   driver.Transfer,
			SenderEID:    "alice",
			RecipientEID: "alice",
			TokenType:    "TOK",
			Amount:       big.NewInt(90),
			Timestamp:    now,
			Status:       driver.Pending,
		},
	}, recs)

	// Issue
	input = simpleTransfer()
	input.Inputs = token.NewInputStream(qsMock{}, []*token.Input{}, 64)
	recs, err = ttxdb.TransactionRecords(&input, now)
	assert.NoError(t, err)
	assert.Equal(t, []driver.TransactionRecord{
		{
			TxID:         input.Anchor,
			ActionType:   driver.Issue,
			SenderEID:    "",
			RecipientEID: "bob",
			TokenType:    "TOK",
			Amount:       big.NewInt(10),
			Timestamp:    now,
			Status:       driver.Pending,
		},
	}, recs)

	// Redeem
	input = redeem()
	recs, err = ttxdb.TransactionRecords(&input, now)
	assert.NoError(t, err)
	assert.Equal(t, []driver.TransactionRecord{
		{
			TxID:         input.Anchor,
			ActionType:   driver.Redeem,
			SenderEID:    "alice",
			RecipientEID: "",
			TokenType:    "TOK",
			Amount:       big.NewInt(10),
			Timestamp:    now,
			Status:       driver.Pending,
		},
	}, recs)
}

func TestMovementRecords(t *testing.T) {
	now := time.Now()

	// Transfer
	input := simpleTransfer()
	recs, err := ttxdb.Movements(&input, now)
	assert.NoError(t, err)
	assert.Equal(t, []driver.MovementRecord{
		{
			TxID:         input.Anchor,
			EnrollmentID: "alice",
			TokenType:    "TOK",
			Amount:       big.NewInt(-10),
			Timestamp:    now,
			Status:       driver.Pending,
		},
		{
			TxID:         input.Anchor,
			EnrollmentID: "bob",
			TokenType:    "TOK",
			Amount:       big.NewInt(10),
			Timestamp:    now,
			Status:       driver.Pending,
		},
	}, recs)

	input = transferWithChange()
	recs, err = ttxdb.Movements(&input, now)
	assert.NoError(t, err)
	assert.Equal(t, []driver.MovementRecord{
		{
			TxID:         input.Anchor,
			EnrollmentID: "alice",
			TokenType:    "TOK",
			Amount:       big.NewInt(-10),
			Timestamp:    now,
			Status:       driver.Pending,
		},
		{
			TxID:         input.Anchor,
			EnrollmentID: "bob",
			TokenType:    "TOK",
			Amount:       big.NewInt(10),
			Timestamp:    now,
			Status:       driver.Pending,
		},
	}, recs)

	// Issue
	input = simpleTransfer()
	input.Inputs = token.NewInputStream(qsMock{}, []*token.Input{}, 64)
	recs, err = ttxdb.Movements(&input, now)
	assert.NoError(t, err)
	assert.Equal(t, []driver.MovementRecord{
		{
			TxID:         input.Anchor,
			EnrollmentID: "bob",
			TokenType:    "TOK",
			Amount:       big.NewInt(10),
			Timestamp:    now,
			Status:       driver.Pending,
		},
	}, recs)

	// Redeem
	input = redeem()
	recs, err = ttxdb.Movements(&input, now)
	assert.NoError(t, err)
	assert.Equal(t, []driver.MovementRecord{
		{
			TxID:         input.Anchor,
			EnrollmentID: "alice",
			TokenType:    "TOK",
			Amount:       big.NewInt(-10),
			Timestamp:    now,
			Status:       driver.Pending,
		},
	}, recs)
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

func transferWithChange() token.AuditRecord {
	input1 := &token.Input{
		ActionIndex:  0,
		EnrollmentID: "alice",
		Type:         "TOK",
		Quantity:     token2.NewQuantityFromUInt64(100),
	}
	output1 := &token.Output{
		ActionIndex:  0,
		EnrollmentID: "bob",
		Type:         "TOK",
		Quantity:     token2.NewQuantityFromUInt64(10),
	}
	output2 := &token.Output{
		ActionIndex:  0,
		EnrollmentID: "alice",
		Type:         "TOK",
		Quantity:     token2.NewQuantityFromUInt64(90),
	}
	return token.AuditRecord{
		Anchor:  "test",
		Inputs:  token.NewInputStream(qsMock{}, []*token.Input{input1}, 64),
		Outputs: token.NewOutputStream([]*token.Output{output1, output2}, 64),
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
