/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package auditdb

import (
	"math/big"

	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditor/auditdb/driver"
)

type PaymentsFilter struct {
	db *AuditDB

	EnrollmentIds  []string
	Types          []string
	LastNumRecords int

	records []*driver.Record
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
	records, err := f.db.db.Query(f.EnrollmentIds, f.Types, nil, driver.FromLast, driver.Sent, f.LastNumRecords)
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
	db *AuditDB

	EnrollmentIds []string
	Types         []string

	records []*driver.Record
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
	records, err := f.db.db.Query(f.EnrollmentIds, f.Types, nil, driver.FromBeginning, driver.All, 0)
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
