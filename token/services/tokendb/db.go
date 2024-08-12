/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tokendb

import (
	"context"
	"reflect"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/pkg/errors"
)

type (
	Holder  = db.DriverHolder[*DB, driver.TokenDB, driver.TokenDBDriver]
	Manager = db.Manager[*DB, driver.TokenDB, driver.TokenDBDriver]

	NotifierHolder  = db.DriverHolder[*Notifier, driver.TokenNotifier, driver.TokenNotifierDriver]
	NotifierManager = db.Manager[*Notifier, driver.TokenNotifier, driver.TokenNotifierDriver]
)

type Notifier struct {
	driver.TokenNotifier
}

func NewNotifierHolder(drivers []db.NamedDriver[driver.TokenNotifierDriver]) *NotifierHolder {
	return db.NewDriverHolder(func(p driver.TokenNotifier) *Notifier { return &Notifier{p} }, drivers...)
}

var managerType = reflect.TypeOf((*Manager)(nil))

func NewHolder(drivers []db.NamedDriver[driver.TokenDBDriver]) *Holder {
	return db.NewDriverHolder(newDB, drivers...)
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

func (d *DB) NewTransaction(ctx context.Context) (*Transaction, error) {
	tx, err := d.TokenDB.NewTokenDBTransaction(ctx)
	if err != nil {
		return nil, err
	}
	return &Transaction{TokenDBTransaction: tx}, nil
}

func newDB(p driver.TokenDB) *DB {
	return &DB{
		TokenDB: p,
	}
}
