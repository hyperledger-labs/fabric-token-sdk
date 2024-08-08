/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"database/sql"
	"runtime/debug"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracing"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/pkg/errors"
)

const (
	QueryLabel      tracing.LabelName = "query"
	ResultRowsLabel tracing.LabelName = "result_rows"
)

var logger = logging.MustGetLogger("token-sdk.sql")

func QueryUnique[T any](db *sql.DB, query string, args ...any) (T, error) {
	logger.Debug(query, args)
	row := db.QueryRow(query, args...)
	var result T
	var err error
	if err = row.Scan(&result); err != nil && errors.Is(err, sql.ErrNoRows) {
		return result, nil
	}
	return result, err
}

func initSchema(db *sql.DB, schemas ...string) (err error) {
	logger.Info("creating tables")
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil && tx != nil {
			if err := tx.Rollback(); err != nil {
				logger.Errorf("failed to rollback [%s][%s]", err, debug.Stack())
			}
		}
	}()
	for _, schema := range schemas {
		logger.Debug(schema)
		if _, err = tx.Exec(schema); err != nil {
			return errors.Wrap(err, "error creating schema")
		}
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	return
}
