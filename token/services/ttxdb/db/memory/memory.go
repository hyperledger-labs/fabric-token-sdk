/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package memory

import (
	"crypto/sha256"
	"fmt"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	sqldb "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb/db/sql"
	"github.com/pkg/errors"
	_ "modernc.org/sqlite"
)

type Driver struct {
	*sql.Driver
}

func NewDriver() *Driver {
	return &Driver{Driver: sql.NewDriver()}
}

// Open returns a pure go sqlite implementation in memory for testing purposes.
func (d *Driver) Open(_ driver.ConfigProvider, tmsID token.TMSID) (driver.TokenTransactionDB, error) {
	h := sha256.New()
	if _, err := h.Write([]byte(tmsID.String())); err != nil {
		return nil, err
	}

	sqlDB, err := d.Driver.OpenSQLDB(
		"sqlite",
		fmt.Sprintf("file:%x?mode=memory&cache=shared", h.Sum(nil)),
		10,
		false,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open memory db for [%s]", tmsID)
	}

	return sqldb.NewTransactionDB(sqlDB, "memory", true)
}

func init() {
	ttxdb.Register("memory", NewDriver())
}
