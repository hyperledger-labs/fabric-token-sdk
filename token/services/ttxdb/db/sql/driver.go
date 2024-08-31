/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql"
	common2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/postgres"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/sqlite"
)

const optsKey = "ttxdb.persistence.opts"

func NewDriver() db.NamedDriver[driver.TTXDBDriver] {
	return db.NamedDriver[driver.TTXDBDriver]{
		Name: sql.SQLPersistence,
		Driver: common.NewOpenerFromMap(optsKey, map[common2.SQLDriverType]common.OpenDBFunc[driver.TokenTransactionDB]{
			sql.SQLite:   sqlite.OpenTransactionDB,
			sql.Postgres: postgres.OpenTransactionDB,
		}),
	}
}
