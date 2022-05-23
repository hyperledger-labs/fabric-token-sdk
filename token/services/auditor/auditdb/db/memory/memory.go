/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package memory

import (
	"time"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditor/auditdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditor/auditdb/driver"
)

type Persistence struct {
	movementRecords    []*driver.MovementRecord
	transactionRecords []*driver.TransactionRecord
}

func (p *Persistence) QueryMovements(ids []string, types []string, status []driver.TxStatus, direction driver.SearchDirection, value driver.MovementDirection, numRecords int) ([]*driver.MovementRecord, error) {
	var res []*driver.MovementRecord

	var cursor int
	switch direction {
	case driver.FromBeginning:
		cursor = -1
	case driver.FromLast:
		cursor = len(p.movementRecords)
	default:
		panic("direction not valid")
	}
	counter := 0
	for {
		switch direction {
		case driver.FromBeginning:
			cursor++
		case driver.FromLast:
			cursor--
		}
		if cursor < 0 || cursor >= len(p.movementRecords) {
			break
		}

		record := p.movementRecords[cursor]
		if len(ids) != 0 {
			found := false
			for _, id := range ids {
				if record.EnrollmentID == id {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		if len(types) != 0 {
			found := false
			for _, typ := range types {
				if record.TokenType == typ {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		if len(status) != 0 {
			found := false
			for _, st := range status {
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

		if value == driver.Sent && record.Amount.Sign() > 0 {
			continue
		}
		if value == driver.Received && record.Amount.Sign() < 0 {
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

func (d Driver) Open(sp view2.ServiceProvider, name string) (driver.AuditDB, error) {
	return &Persistence{}, nil
}

func init() {
	auditdb.Register("memory", &Driver{})
}
