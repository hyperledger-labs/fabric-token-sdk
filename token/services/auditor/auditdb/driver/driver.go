/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package driver

import (
	"math/big"

	view "github.com/hyperledger-labs/fabric-smart-client/platform/view"
)

type Direction int

const (
	FromLast Direction = iota
	FromBeginning
)

type Value int

const (
	Sent Value = iota
	Received
	All
)

type Status string

const (
	Pending   Status = "Pending"
	Confirmed Status = "Confirmed"
	Deleted   Status = "Deleted"
)

type Record struct {
	TxID         string
	ActionIndex  uint32
	EnrollmentID string
	Type         string
	// Positive is money received. Negative is money sent
	Amount *big.Int
	Status Status
}

type AuditDB interface {
	Close() error
	BeginUpdate() error
	Commit() error
	Discard() error
	AddRecord(record *Record) error
	SetStatus(txID string, status Status) error
	Query(ids []string, types []string, status []Status, direction Direction, value Value, numRecords int) ([]*Record, error)
}

type Driver interface {
	Open(sp view.ServiceProvider, name string) (AuditDB, error)
}
