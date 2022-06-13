/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package memory

import (
	"math/big"
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb/driver"

	"github.com/stretchr/testify/assert"
)

func Test(t *testing.T) {
	db := &Persistence{}
	err := db.AddMovement(&driver.MovementRecord{
		TxID:         "0",
		EnrollmentID: "alice",
		Amount:       big.NewInt(10),
		TokenType:    "EUR",
		Status:       driver.Pending,
	})
	assert.NoError(t, err)
	err = db.AddMovement(&driver.MovementRecord{
		TxID:         "1",
		EnrollmentID: "alice",
		Amount:       big.NewInt(20),
		TokenType:    "EUR",
		Status:       driver.Pending,
	})
	assert.NoError(t, err)
	err = db.AddMovement(&driver.MovementRecord{
		TxID:         "2",
		EnrollmentID: "alice",
		Amount:       big.NewInt(-5),
		TokenType:    "EUR",
		Status:       driver.Pending,
	})
	assert.NoError(t, err)

	records, err := db.QueryMovements(driver.QueryMovementsParams{EnrollmentIDs: []string{"alice"}, TokenTypes: []string{"EUR"}, SearchDirection: driver.FromBeginning, MovementDirection: driver.Sent})
	assert.NoError(t, err)
	assert.Len(t, records, 1)
	records, err = db.QueryMovements(driver.QueryMovementsParams{EnrollmentIDs: []string{"alice"}, TokenTypes: []string{"EUR"}, SearchDirection: driver.FromLast, MovementDirection: driver.Sent})
	assert.NoError(t, err)
	assert.Len(t, records, 1)
	records, err = db.QueryMovements(driver.QueryMovementsParams{EnrollmentIDs: []string{"alice"}, TokenTypes: []string{"EUR"}, SearchDirection: driver.FromLast, MovementDirection: driver.Received})
	assert.NoError(t, err)
	assert.Len(t, records, 2)
	records, err = db.QueryMovements(driver.QueryMovementsParams{EnrollmentIDs: []string{"alice"}, TokenTypes: []string{"EUR"}, SearchDirection: driver.FromLast, MovementDirection: driver.Received, NumRecords: 1})
	assert.NoError(t, err)
	assert.Len(t, records, 1)

	records, err = db.QueryMovements(driver.QueryMovementsParams{EnrollmentIDs: []string{"bob"}, TokenTypes: []string{"EUR"}, SearchDirection: driver.FromBeginning, MovementDirection: driver.Sent})
	assert.NoError(t, err)
	assert.Len(t, records, 0)
	records, err = db.QueryMovements(driver.QueryMovementsParams{EnrollmentIDs: []string{"alice"}, TokenTypes: []string{"USD"}, SearchDirection: driver.FromBeginning, MovementDirection: driver.Sent})
	assert.NoError(t, err)
	assert.Len(t, records, 0)
	records, err = db.QueryMovements(driver.QueryMovementsParams{EnrollmentIDs: []string{"alice"}, TokenTypes: []string{"EUR"}, TxStatuses: []driver.TxStatus{driver.Confirmed}, SearchDirection: driver.FromBeginning, MovementDirection: driver.Sent})
	assert.NoError(t, err)
	assert.Len(t, records, 0)
}
