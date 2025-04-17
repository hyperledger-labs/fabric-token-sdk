/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tokendb

import (
	"reflect"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/pkg/errors"
)

type (
	Manager         = db.Manager[*DB]
	NotifierManager = db.Manager[*Notifier]
)

type Notifier struct {
	driver.TokenNotifier
}

var managerType = reflect.TypeOf((*Manager)(nil))

func NewNotifierManager(dh *db.DriverHolder) *NotifierManager {
	return db.MappedManager[driver.TokenNotifier, *Notifier](dh.NewTokenNotifierManager(), func(p driver.TokenNotifier) (*Notifier, error) { return &Notifier{p}, nil })
}

func NewManager(dh *db.DriverHolder) *Manager {
	return db.MappedManager[driver.TokenDB, *DB](dh.NewTokenManager(), newDB)
}

func GetByTMSId(sp token.ServiceProvider, tmsID token.TMSID) (*DB, error) {
	s, err := sp.GetService(managerType)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get manager service")
	}
	c, err := s.(*Manager).DBByTMSId(tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get db for tms [%s]", tmsID)
	}
	return c, nil
}

type TokenRecord = driver.TokenRecord

type Transaction struct {
	driver.TokenDBTransaction
}

// DB is a database that stores token transactions related information
type DB struct {
	driver.TokenDB
}

func (d *DB) NewTransaction() (*Transaction, error) {
	tx, err := d.TokenDB.NewTokenDBTransaction()
	if err != nil {
		return nil, err
	}
	return &Transaction{TokenDBTransaction: tx}, nil
}

func newDB(p driver.TokenDB) (*DB, error) {
	return &DB{TokenDB: p}, nil
}
