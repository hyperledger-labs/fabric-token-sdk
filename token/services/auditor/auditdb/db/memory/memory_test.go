/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package memory

import (
	"math/big"
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditor/auditdb/driver"
	"github.com/stretchr/testify/assert"
)

func Test(t *testing.T) {
	db := &Persistence{}
	err := db.AddRecord(&driver.Record{
		TxID:         "0",
		ActionIndex:  0,
		EnrollmentID: "alice",
		Amount:       big.NewInt(10),
		Type:         "EUR",
		Status:       driver.Pending,
	})
	assert.NoError(t, err)
	err = db.AddRecord(&driver.Record{
		TxID:         "0",
		ActionIndex:  0,
		EnrollmentID: "alice",
		Amount:       big.NewInt(20),
		Type:         "EUR",
		Status:       driver.Pending,
	})
	assert.NoError(t, err)
	err = db.AddRecord(&driver.Record{
		TxID:         "0",
		ActionIndex:  0,
		EnrollmentID: "alice",
		Amount:       big.NewInt(-5),
		Type:         "EUR",
		Status:       driver.Pending,
	})
	assert.NoError(t, err)

	records, err := db.Query([]string{"alice"}, []string{"EUR"}, nil, driver.FromBeginning, driver.Sent, 0)
	assert.NoError(t, err)
	assert.Len(t, records, 1)
	records, err = db.Query([]string{"alice"}, []string{"EUR"}, nil, driver.FromLast, driver.Sent, 0)
	assert.NoError(t, err)
	assert.Len(t, records, 1)
	records, err = db.Query([]string{"alice"}, []string{"EUR"}, nil, driver.FromLast, driver.Received, 0)
	assert.NoError(t, err)
	assert.Len(t, records, 2)
	records, err = db.Query([]string{"alice"}, []string{"EUR"}, nil, driver.FromLast, driver.Received, 1)
	assert.NoError(t, err)
	assert.Len(t, records, 1)

	records, err = db.Query([]string{"bob"}, []string{"EUR"}, nil, driver.FromBeginning, driver.Sent, 0)
	assert.NoError(t, err)
	assert.Len(t, records, 0)
	records, err = db.Query([]string{"alice"}, []string{"USD"}, nil, driver.FromBeginning, driver.Sent, 0)
	assert.NoError(t, err)
	assert.Len(t, records, 0)
	records, err = db.Query([]string{"alice"}, []string{"EUR"}, []driver.Status{driver.Confirmed}, driver.FromBeginning, driver.Sent, 0)
	assert.NoError(t, err)
	assert.Len(t, records, 0)
}
