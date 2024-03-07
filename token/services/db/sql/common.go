/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"database/sql"
	"fmt"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/pkg/errors"
)

func DatasourceName(tmsID token.TMSID) string {
	return fmt.Sprintf("%s-%s-%s", tmsID.Network, tmsID.Channel, tmsID.Namespace)
}

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
