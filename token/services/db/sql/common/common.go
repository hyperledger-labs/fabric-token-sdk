/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"database/sql"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/common"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracing"
)

const (
	QueryLabel      tracing.LabelName = "query"
	ResultRowsLabel tracing.LabelName = "result_rows"
)

type Opts = common.Opts

type NewDBFunc[T any] func(readDB, writeDB *sql.DB, opts NewDBOpts) (T, error)

type NewDBOpts struct {
	DataSource   string
	TablePrefix  string
	CreateSchema bool
}

func NewDBOptsFromOpts(o Opts) NewDBOpts {
	return NewDBOpts{
		DataSource:   o.DataSource,
		TablePrefix:  o.TablePrefix,
		CreateSchema: !o.SkipCreateTable,
	}
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
