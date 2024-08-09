/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package db

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token"
)

type OpenFunc[D any] func(cp ConfigProvider, tmsID token.TMSID) (D, error)

type SQLDriver[D any] struct {
	f OpenFunc[D]
}

func NewSQLDriver[D any](f OpenFunc[D]) *SQLDriver[D] {
	return &SQLDriver[D]{f: f}
}

func (d *SQLDriver[D]) Open(cp ConfigProvider, tmsID token.TMSID) (D, error) {
	return d.f(cp, tmsID)
}
