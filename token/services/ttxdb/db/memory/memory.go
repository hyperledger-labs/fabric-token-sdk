/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package memory

import (
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb/driver"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
)

type Persistence struct {
	movementRecords    []*driver.MovementRecord
	transactionRecords []*driver.TransactionRecord
}

func (p *Persistence) QueryMovements(enrollmentIDs []string, tokenTypes []string, txStatuses []driver.TxStatus, searchDirection driver.SearchDirection, movementDirection driver.MovementDirection, numRecords int) ([]*driver.MovementRecord, error) {
	var res []*driver.MovementRecord

	var cursor int
	switch searchDirection {
	case driver.FromBeginning:
		cursor = -1
	case driver.FromLast:
		cursor = len(p.movementRecords)
	default:
		panic("direction not valid")
	}
	counter := 0
	for {
		switch searchDirection {
		case driver.FromBeginning:
			cursor++
		case driver.FromLast:
			cursor--
		}
		if cursor < 0 || cursor >= len(p.movementRecords) {
			break
		}

		record := p.movementRecords[cursor]
		if len(enrollmentIDs) != 0 {
			found := false
			for _, id := range enrollmentIDs {
				if record.EnrollmentID == id {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		if len(tokenTypes) != 0 {
			found := false
			for _, typ := range tokenTypes {
				if record.TokenType == typ {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		if len(txStatuses) != 0 {
			found := false
			for _, st := range txStatuses {
				if record.Status == st {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		} else {
			// exclude the deleted
			if record.Status == driver.Deleted {
				continue
			}
		}

		if numRecords != 0 && counter+1 > numRecords {
			break
		}

		if movementDirection == driver.Sent && record.Amount.Sign() > 0 {
			continue
		}
		if movementDirection == driver.Received && record.Amount.Sign() < 0 {
			continue
		}

		counter++
		res = append(res, record)
	}

	return res, nil
}

func (p *Persistence) AddMovement(record *driver.MovementRecord) error {
	p.movementRecords = append(p.movementRecords, record)

	return nil
}

func (p *Persistence) QueryTransactions(from, to *time.Time) (driver.TransactionIterator, error) {
	// search over the transaction for those whose timestamp is between from and to
	var subset []*driver.TransactionRecord
	for _, record := range p.transactionRecords {
		if from != nil && record.Timestamp.Before(*from) {
			continue
		}
		if to != nil && record.Timestamp.After(*to) {
			continue
		}
		subset = append(subset, record)
	}
	return &TransactionIterator{txs: subset}, nil
}

func (p *Persistence) AddTransaction(record *driver.TransactionRecord) error {
	p.transactionRecords = append(p.transactionRecords, record)

	return nil
}

func (p *Persistence) SetStatus(txID string, status driver.TxStatus) error {
	// movements
	for _, record := range p.movementRecords {
		if record.TxID == txID {
			record.Status = status
		}
	}
	// transactions
	for _, record := range p.transactionRecords {
		if record.TxID == txID {
			record.Status = status
		}
	}
	return nil
}

func (p *Persistence) Close() error {
	return nil
}

func (p *Persistence) BeginUpdate() error {
	return nil
}

func (p *Persistence) Commit() error {
	return nil
}

func (p *Persistence) Discard() error {
	return nil
}

type TransactionIterator struct {
	txs    []*driver.TransactionRecord
	cursor int
}

func (t *TransactionIterator) Close() {
}

func (t *TransactionIterator) Next() (*driver.TransactionRecord, error) {
	// return next transaction, if any
	if t.cursor >= len(t.txs) {
		return nil, nil
	}
	record := t.txs[t.cursor]
	t.cursor++
	return record, nil
}

type Driver struct{}

func (d Driver) Open(sp view2.ServiceProvider, name string) (driver.DB, error) {
	return &Persistence{}, nil
}

func init() {
	ttxdb.Register("memory", &Driver{})
}
