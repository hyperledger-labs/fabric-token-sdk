/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tokendb

import (
	"reflect"

	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/config"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/multiplexed"
	"github.com/pkg/errors"
)

type (
	StoreServiceManager db.StoreServiceManager[*StoreService]
	NotifierManager     db.StoreServiceManager[*Notifier]
)

type Notifier struct {
	driver.TokenNotifier
}

var managerType = reflect.TypeOf((*StoreServiceManager)(nil))

func NewNotifierManager(cp driver2.ConfigService, drivers multiplexed.Driver) NotifierManager {
	return db.NewStoreServiceManager(config.NewService(cp), "tokendb.persistence", drivers.NewTokenNotifier, func(p driver.TokenNotifier) (*Notifier, error) { return &Notifier{p}, nil })
}

func NewStoreServiceManager(cp driver2.ConfigService, drivers multiplexed.Driver) StoreServiceManager {
	return db.NewStoreServiceManager(config.NewService(cp), "tokendb.persistence", drivers.NewToken, newStoreService)
}

func GetByTMSId(sp token.ServiceProvider, tmsID token.TMSID) (*StoreService, error) {
	s, err := sp.GetService(managerType)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get manager service")
	}
	c, err := s.(StoreServiceManager).StoreServiceByTMSId(tmsID)
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
	tx, err := d.NewTokenDBTransaction()
	if err != nil {
		return nil, err
	}
	return &Transaction{TokenStoreTransaction: tx}, nil
}

func newStoreService(p driver.TokenStore) (*StoreService, error) {
	return &StoreService{TokenStore: p}, nil
}
