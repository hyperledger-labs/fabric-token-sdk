/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"database/sql"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracing"
	"github.com/pkg/errors"
)

const (
	QueryLabel      tracing.LabelName = "query"
	ResultRowsLabel tracing.LabelName = "result_rows"
)

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

type Closer interface {
	Close() error
}

func Close(closer Closer) {
	if closer == nil {
		return
	}
	if err := closer.Close(); err != nil {
		logger.Errorf("failed closing connection: %s", err)
	}
}
