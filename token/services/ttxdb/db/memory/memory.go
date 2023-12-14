/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package memory

import (
	"crypto/sha256"
	"fmt"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb/db/sql"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb/driver"
	_ "modernc.org/sqlite"
)

type Driver struct{}

// This database driver runs a pure go sqlite implementation in memory for testing purposes.
func (d Driver) Open(sp view2.ServiceProvider, name string) (driver.TokenTransactionDB, error) {
	h := sha256.New()
	if _, err := h.Write([]byte(name)); err != nil {
		return nil, err
	}

	return sql.OpenDB("sqlite", fmt.Sprintf("file:%x?mode=memory&cache=shared", h.Sum(nil)), "test", name, true)
}

func init() {
	ttxdb.Register("memory", &Driver{})
}
