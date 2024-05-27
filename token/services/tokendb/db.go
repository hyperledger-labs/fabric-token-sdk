/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tokendb

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
)

var (
	holder = db.NewDriverHolder[*DB, driver.TokenDB, driver.TokenDBDriver](newDB)
)

func Register(name string, driver driver.TokenDBDriver) { holder.Register(name, driver) }

func Drivers() []string { return holder.DriverNames() }

type Manager = db.Manager[*DB, driver.TokenDB, driver.TokenDBDriver]

func NewManager(cp driver.ConfigProvider, config db.Config) *Manager {
	return holder.NewManager(cp, config)
}

func GetByTMSId(sp token.ServiceProvider, tmsID token.TMSID) (*DB, error) {
	return holder.GetByTMSId(sp, tmsID)
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

func newDB(p driver.TokenDB) *DB {
	return &DB{
		TokenDB: p,
	}
}
