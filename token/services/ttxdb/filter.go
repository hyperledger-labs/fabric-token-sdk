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

type PaymentsFilter struct {
	db *DB

	EnrollmentIds  []string
	Types          []string
	LastNumRecords int

	records []*driver.MovementRecord
}

func (f *PaymentsFilter) ByEnrollmentId(id string) *PaymentsFilter {
	f.EnrollmentIds = append(f.EnrollmentIds, id)
	return f
}

func (f *PaymentsFilter) ByType(tokenType string) *PaymentsFilter {
	f.Types = append(f.Types, tokenType)
	return f
}

func (f *PaymentsFilter) Last(num int) *PaymentsFilter {
	f.LastNumRecords = num
	return f
}

func (f *PaymentsFilter) Execute() (*PaymentsFilter, error) {
	records, err := f.db.db.QueryMovements(
		f.EnrollmentIds,
		f.Types,
		[]driver.TxStatus{driver.Pending, driver.Confirmed},
		driver.FromLast,
		driver.Sent,
		f.LastNumRecords,
	)
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
	db *DB

	EnrollmentIds []string
	Types         []string

	records []*driver.MovementRecord
}

func (f *HoldingsFilter) ByEnrollmentId(id string) *HoldingsFilter {
	f.EnrollmentIds = append(f.EnrollmentIds, id)
	return f
}

func (f *HoldingsFilter) ByType(tokenType string) *HoldingsFilter {
	f.Types = append(f.Types, tokenType)
	return f
}

func (f *HoldingsFilter) Execute() (*HoldingsFilter, error) {
	records, err := f.db.db.QueryMovements(f.EnrollmentIds, f.Types, []driver.TxStatus{driver.Pending, driver.Confirmed}, driver.FromBeginning, driver.All, 0)
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
