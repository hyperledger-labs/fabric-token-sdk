/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package memory

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb/driver"
	"github.com/pkg/errors"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
)

type Persistence struct {
	movementRecords    []*driver.MovementRecord
	transactionRecords []*driver.TransactionRecord
}

func (p *Persistence) QueryMovements(params driver.QueryMovementsParams) ([]*driver.MovementRecord, error) {
	var res []*driver.MovementRecord

	var cursor int
	switch params.SearchDirection {
	case driver.FromBeginning:
		cursor = -1
	case driver.FromLast:
		cursor = len(p.movementRecords)
	default:
		return nil, errors.Errorf("direction not valid")
	}
	counter := 0
	for {
		switch params.SearchDirection {
		case driver.FromBeginning:
			cursor++
		case driver.FromLast:
			cursor--
		}
		if cursor < 0 || cursor >= len(p.movementRecords) {
			break
		}

		record := p.movementRecords[cursor]
		if len(params.EnrollmentIDs) != 0 {
			found := false
			for _, id := range params.EnrollmentIDs {
				if record.EnrollmentID == id {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		if len(params.TokenTypes) != 0 {
			found := false
			for _, typ := range params.TokenTypes {
				if record.TokenType == typ {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		if len(params.TxStatuses) != 0 {
			found := false
			for _, st := range params.TxStatuses {
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

		if params.NumRecords != 0 && counter+1 > params.NumRecords {
			break
		}

		if params.MovementDirection == driver.Sent && record.Amount.Sign() > 0 {
			continue
		}
		if params.MovementDirection == driver.Received && record.Amount.Sign() < 0 {
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

func (p *Persistence) QueryTransactions(params driver.QueryTransactionsParams) (driver.TransactionIterator, error) {
	// search over the transaction for those whose timestamp is between from and to
	var subset []*driver.TransactionRecord
	for _, record := range p.transactionRecords {
		if params.From != nil && record.Timestamp.Before(*params.From) {
			continue
		}
		if params.To != nil && record.Timestamp.After(*params.To) {
			continue
		}
		if len(params.SenderWallet) != 0 && record.SenderEID != params.SenderWallet {
			continue
		}
		if len(params.ActionTypes) != 0 {
			found := false
			for _, actionType := range params.ActionTypes {
				if actionType == record.ActionType {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		if len(params.Statuses) != 0 {
			found := false
			for _, statusType := range params.Statuses {
				if statusType == record.Status {
					found = true
					break
				}
			}
			if !found {
				continue
			}
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

func (p *Persistence) GetStatus(txID string) (driver.TxStatus, error) {
	// transactions
	for _, record := range p.transactionRecords {
		if record.TxID == txID {
			return record.Status, nil
		}
	}
	return driver.Unknown, errors.Errorf("transaction [%s] not found", txID)
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

func (d Driver) Open(sp view2.ServiceProvider, name string) (driver.TokenTransactionDB, error) {
	return &Persistence{}, nil
}

func init() {
	ttxdb.Register("memory", &Driver{})
}
