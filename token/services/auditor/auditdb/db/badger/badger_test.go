/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package badger

import (
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditor/auditdb/driver"
)

func TestMovements(t *testing.T) {
	dbpath := filepath.Join(tempDir, "DB-TestRangeQueries")
	db, err := OpenDB(dbpath)
	defer db.Close()
	assert.NoError(t, err)
	assert.NotNil(t, db)

	assert.NoError(t, db.BeginUpdate())
	err = db.AddMovement(&driver.MovementRecord{
		TxID:         "0",
		EnrollmentID: "alice",
		TokenType:    "magic",
		Amount:       big.NewInt(10),
		Status:       driver.Pending,
	})
	assert.NoError(t, err)
	err = db.AddMovement(&driver.MovementRecord{
		TxID:         "1",
		EnrollmentID: "alice",
		TokenType:    "magic",
		Amount:       big.NewInt(20),
		Status:       driver.Pending,
	})
	assert.NoError(t, err)
	err = db.AddMovement(&driver.MovementRecord{
		TxID:         "2",
		EnrollmentID: "alice",
		TokenType:    "magic",
		Amount:       big.NewInt(30),
		Status:       driver.Pending,
	})
	assert.NoError(t, err)
	assert.NoError(t, db.Commit())

	records, err := db.QueryMovements(nil, nil, []driver.TxStatus{driver.Pending}, driver.FromLast, driver.Received, 2)
	assert.NoError(t, err)
	assert.Len(t, records, 2)

	records, err = db.QueryMovements(nil, nil, []driver.TxStatus{driver.Pending}, driver.FromLast, driver.Received, 3)
	assert.NoError(t, err)
	assert.Len(t, records, 3)

	assert.NoError(t, db.BeginUpdate())
	assert.NoError(t, db.SetStatus("2", driver.Confirmed))
	assert.NoError(t, db.Commit())

	records, err = db.QueryMovements(nil, nil, []driver.TxStatus{driver.Pending}, driver.FromLast, driver.Received, 3)
	assert.NoError(t, err)
	assert.Len(t, records, 2)
}

func TestTransaction(t *testing.T) {
	dbpath := filepath.Join(tempDir, "DB-TestRangeQueries")
	db, err := OpenDB(dbpath)
	defer db.Close()
	assert.NoError(t, err)
	assert.NotNil(t, db)

	assert.NoError(t, db.BeginUpdate())
	now := time.Now().UTC()
	tr := &driver.TransactionRecord{
		TxID:            "0",
		TransactionType: driver.Issue,
		SenderEID:       "",
		RecipientEID:    "alice",
		TokenType:       "magic",
		Amount:          big.NewInt(10),
		Timestamp:       now,
		Status:          driver.Pending,
	}
	assert.NoError(t, db.AddTransaction(tr))
	assert.NoError(t, db.Commit())

	it, err := db.QueryTransactions(&now, &now)
	assert.NoError(t, err)
	tr2, err := it.Next()
	assert.NoError(t, err)
	assert.Equal(t, tr, tr2)
	it.Close()
}

var tempDir string

func TestMain(m *testing.M) {
	var err error
	tempDir, err = ioutil.TempDir("", "badger-fsc-test")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temporary directory: %v", err)
		os.Exit(-1)
	}
	defer os.RemoveAll(tempDir)

	m.Run()
}
