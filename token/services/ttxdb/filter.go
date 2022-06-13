/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttxdb

import (
	"math/big"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb/driver"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

// PaymentsFilter is a filter for payments.
type PaymentsFilter struct {
	db      *DB
	params  driver.QueryMovementsParams
	records []*driver.MovementRecord
}

// ByEnrollmentId add an enrollment id to the filter.
func (f *PaymentsFilter) ByEnrollmentId(id string) *PaymentsFilter {
	f.params.EnrollmentIDs = append(f.params.EnrollmentIDs, id)
	return f
}

func (f *PaymentsFilter) ByType(tokenType string) *PaymentsFilter {
	f.params.TokenTypes = append(f.params.TokenTypes, tokenType)
	return f
}

func (f *PaymentsFilter) Last(num int) *PaymentsFilter {
	f.params.NumRecords = num
	return f
}

func (f *PaymentsFilter) Execute() (*PaymentsFilter, error) {
	f.params.TxStatuses = []driver.TxStatus{driver.Pending, driver.Confirmed}
	f.params.MovementDirection = driver.Sent
	f.params.SearchDirection = driver.FromLast
	records, err := f.db.db.QueryMovements(f.params)
	if err != nil {
		return nil, err
	}
	f.records = records
	return f, nil
}

func (f *PaymentsFilter) Sum() token2.Quantity {
	sum := big.NewInt(0)
	for _, record := range f.records {
		sum = sum.Add(sum, record.Amount)
	}
	sum.Neg(sum)
	return token2.NewQuantityFromBig64(sum)
}

type HoldingsFilter struct {
	db      *DB
	params  driver.QueryMovementsParams
	records []*driver.MovementRecord
}

func (f *HoldingsFilter) ByEnrollmentId(id string) *HoldingsFilter {
	f.params.EnrollmentIDs = append(f.params.EnrollmentIDs, id)
	return f
}

func (f *HoldingsFilter) ByType(tokenType string) *HoldingsFilter {
	f.params.TokenTypes = append(f.params.TokenTypes, tokenType)
	return f
}

func (f *HoldingsFilter) Execute() (*HoldingsFilter, error) {
	f.params.TxStatuses = []driver.TxStatus{driver.Pending, driver.Confirmed}
	f.params.MovementDirection = driver.All
	f.params.SearchDirection = driver.FromBeginning
	records, err := f.db.db.QueryMovements(f.params)
	if err != nil {
		return nil, err
	}
	f.records = records
	return f, nil
}

func (f *HoldingsFilter) Sum() token2.Quantity {
	sum := big.NewInt(0)
	for _, record := range f.records {
		sum = sum.Add(sum, record.Amount)
	}
	return token2.NewQuantityFromBig64(sum)
}
