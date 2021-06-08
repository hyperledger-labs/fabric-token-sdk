/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package memory

import (
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditor/auditdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/auditor/auditdb/driver"
)

type Persistence struct {
	records []*driver.Record
}

func (p *Persistence) Query(ids []string, types []string, status []driver.Status, direction driver.Direction, value driver.Value, numRecords int) ([]*driver.Record, error) {
	var res []*driver.Record

	var cursor int
	switch direction {
	case driver.FromBeginning:
		cursor = -1
	case driver.FromLast:
		cursor = len(p.records)
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
		if cursor < 0 || cursor >= len(p.records) {
			break
		}

		record := p.records[cursor]
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
				if record.Type == typ {
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

func (p *Persistence) AddRecord(record *driver.Record) error {
	p.records = append(p.records, record)

	return nil
}

func (p *Persistence) SetStatus(txID string, status driver.Status) error {
	for _, record := range p.records {
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

type Driver struct {
}

func (d Driver) Open(sp view2.ServiceProvider, name string) (driver.AuditDB, error) {
	return &Persistence{}, nil
}

func init() {
	auditdb.Register("memory", &Driver{})
}
