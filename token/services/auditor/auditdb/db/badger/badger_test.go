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

	"github.com/stretchr/testify/assert"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditor/auditdb/driver"
)

func TestDB(t *testing.T) {
	dbpath := filepath.Join(tempDir, "DB-TestRangeQueries")
	db, err := OpenDB(dbpath)
	defer db.Close()
	assert.NoError(t, err)
	assert.NotNil(t, db)

	db.BeginUpdate()

	err = db.AddRecord(&driver.Record{
		TxID:         "0",
		ActionIndex:  0,
		EnrollmentID: "alice",
		Type:         "magic",
		Amount:       big.NewInt(10),
		Status:       driver.Pending,
	})
	assert.NoError(t, err)
	err = db.AddRecord(&driver.Record{
		TxID:         "0",
		ActionIndex:  0,
		EnrollmentID: "alice",
		Type:         "magic",
		Amount:       big.NewInt(20),
		Status:       driver.Pending,
	})
	assert.NoError(t, err)
	err = db.AddRecord(&driver.Record{
		TxID:         "0",
		ActionIndex:  0,
		EnrollmentID: "alice",
		Type:         "magic",
		Amount:       big.NewInt(30),
		Status:       driver.Pending,
	})
	assert.NoError(t, err)
	db.Commit()

	records, err := db.Query(nil, nil, nil, driver.FromLast, driver.Received, 2)
	assert.NoError(t, err)
	assert.Len(t, records, 2)
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
