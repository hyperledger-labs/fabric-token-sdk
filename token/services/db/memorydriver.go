/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package db

import (
	"crypto/sha256"
	"database/sql"
	"fmt"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	sqldb "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql"
	"github.com/pkg/errors"
)

type NewDBFunc[D any] func(db *sql.DB, createSchema bool) (D, error)

type MemoryDriver[D any] struct {
	dbOpener *sqldb.DBOpener
	newDB    NewDBFunc[D]
}

func NewMemoryDriver[D any](dbOpener *sqldb.DBOpener, newDB NewDBFunc[D]) *MemoryDriver[D] {
	return &MemoryDriver[D]{dbOpener: dbOpener, newDB: newDB}
}

// Open returns a pure go sqlite implementation in memory for testing purposes.
func (d *MemoryDriver[D]) Open(_ driver.ConfigProvider, tmsID token.TMSID) (D, error) {
	h := sha256.New()
	if _, err := h.Write([]byte(tmsID.String())); err != nil {
		return utils.Zero[D](), err
	}

	sqlDB, err := d.dbOpener.OpenSQLDB(
		"sqlite",
		fmt.Sprintf("file:%x?mode=memory&cache=shared", h.Sum(nil)),
		10,
		false,
	)
	if err != nil {
		return utils.Zero[D](), errors.Wrapf(err, "failed to open memory db for [%s]", tmsID)
	}

	return d.newDB(sqlDB, true)
}
