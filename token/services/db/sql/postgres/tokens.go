/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package postgres

import (
	"database/sql"

	common2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/postgres"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
	"github.com/pkg/errors"
)

func NewTokenDB(db *sql.DB, opts common.NewDBOpts) (driver.TokenDB, error) {
	return common.NewTokenDB(db, opts, common.NewTokenInterpreter(postgres.NewInterpreter()))
}

type TokenNotifier struct {
	*postgres.Notifier
}

func NewTokenNotifier(db *sql.DB, opts common.NewDBOpts) (driver.TokenNotifier, error) {
	tables, err := common.GetTableNames(opts.TablePrefix)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get table names")
	}
	notifier := postgres.NewNotifier(db, tables.Tokens, opts.DataSource, postgres.AllOperations, "tx_id", "idx")
	if opts.CreateSchema {
		if err = common2.InitSchema(db, notifier.GetSchema()); err != nil {
			return nil, err
		}
	}
	return notifier, nil
}
