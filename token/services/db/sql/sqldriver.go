/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
)

type OpenFunc[D any] func(cp db.ConfigProvider, tmsID token.TMSID) (D, error)

type Driver[D any] struct {
	f OpenFunc[D]
}

func NewDriver[D any](f OpenFunc[D]) *Driver[D] {
	return &Driver[D]{f: f}
}

func (d *Driver[D]) Open(cp db.ConfigProvider, tmsID token.TMSID) (D, error) {
	return d.f(cp, tmsID)
}
