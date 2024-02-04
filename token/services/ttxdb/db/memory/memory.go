/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package memory

import (
	"crypto/sha256"
	"fmt"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
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
func (d *Driver) Open(sp view2.ServiceProvider, tmsID token.TMSID) (driver.TokenTransactionDB, error) {
	name := fmt.Sprintf("%s-%s-%s", tmsID.Network, tmsID.Channel, tmsID.Namespace)
	h := sha256.New()
	if _, err := h.Write([]byte(name)); err != nil {
		return nil, err
	}

	sqlDB, err := d.Driver.OpenSQLDB(
		"sqlite",
		fmt.Sprintf("file:%x?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&mode=memory&cache=shared", h.Sum(nil)),
		10,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open memory db for [%s]", tmsID)
	}

	return sqldb.NewTransactionDB(sqlDB, "memory", name, true)
}

func init() {
	ttxdb.Register("memory", NewDriver())
}
