/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package memory

import (
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils"
	sql2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
)

type Driver[D any] struct {
	dbOpener func(opts common.Opts) (D, error)
}

func NewDriver[D any](dbOpener func(opts common.Opts) (D, error)) *Driver[D] {
	return &Driver[D]{dbOpener: dbOpener}
}

// Open returns a pure go sqlite implementation in memory for testing purposes.
func (d *Driver[D]) Open(_ driver.ConfigProvider, tmsID token.TMSID) (D, error) {
	h := sha256.New()
	if _, err := h.Write([]byte(tmsID.String())); err != nil {
		return utils.Zero[D](), err
	}

	opts := common.Opts{
		Driver:          sql2.SQLite,
		DataSource:      fmt.Sprintf("file:%x?mode=memory&cache=shared", h.Sum(nil)),
		TablePrefix:     "memory",
		SkipCreateTable: false,
		SkipPragmas:     false,
		MaxOpenConns:    10,
		MaxIdleConns:    common.CopyPtr(10),
		MaxIdleTime:     common.CopyPtr(time.Minute),
	}
	return d.dbOpener(opts)
}
