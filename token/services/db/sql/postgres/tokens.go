/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package postgres

import (
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	common2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/postgres"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
)

func NewTokenDB(k common2.Opts) (driver.TokenDB, error) {
	db, err := postgres.OpenDB(k.DataSource, k.MaxOpenConns)
	if err != nil {
		return nil, err
	}
	return common.NewTokenDB(db, common.NewDBOptsFromOpts(k))
}

type TokenNDB struct {
	*common.TokenDB
	*postgres.Notifier
}

func (db *TokenNDB) GetSchema() string {
	return db.TokenDB.GetSchema() + "\n" + db.Notifier.GetSchema()
}

func NewTokenNDB(k common2.Opts) (driver.TokenNDB, error) {
	db, err := postgres.OpenDB(k.DataSource, k.MaxOpenConns)
	if err != nil {
		return nil, err
	}
	tables, err := common.GetTableNames(k.TablePrefix)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get table names")
	}

	tokenDB := &TokenNDB{
		TokenDB:  common.CreateTokenDB(db, tables),
		Notifier: postgres.NewNotifier(db, tables.Tokens, k.DataSource, postgres.AllOperations, "tx_id", "idx"),
	}
	if !k.SkipCreateTable {
		if err = common2.InitSchema(db, []string{tokenDB.GetSchema()}...); err != nil {
			return nil, err
		}
	}
	return tokenDB, nil
}
