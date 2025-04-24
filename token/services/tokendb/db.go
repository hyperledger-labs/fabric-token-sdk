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
	Manager         = db.Manager[*StoreService]
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
	return db.MappedManager[driver.TokenStore, *StoreService](dh.NewTokenManager(), newStoreService)
}

func GetByTMSId(sp token.ServiceProvider, tmsID token.TMSID) (*StoreService, error) {
	s, err := sp.GetService(managerType)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get manager service")
	}
	c, err := s.(*Manager).ServiceByTMSId(tmsID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get db for tms [%s]", tmsID)
	}
	return c, nil
}

type TokenRecord = driver.TokenRecord

type Transaction struct {
	driver.TokenStoreTransaction
}

// StoreService is a database that stores token transactions related information
type StoreService struct {
	driver.TokenStore
}

func (d *StoreService) NewTransaction() (*Transaction, error) {
	tx, err := d.TokenStore.NewTokenDBTransaction()
	if err != nil {
		return nil, err
	}
	return &Transaction{TokenStoreTransaction: tx}, nil
}

func newStoreService(p driver.TokenStore) (*StoreService, error) {
	return &StoreService{TokenStore: p}, nil
}
