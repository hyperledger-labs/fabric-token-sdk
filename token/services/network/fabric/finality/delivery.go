/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package finality

import (
	"context"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
)

type Delivery struct {
	*fabric.Delivery
}

func (d *Delivery) ScanBlock(background context.Context, callback fabric.BlockCallback) error {
	return d.Delivery.ScanBlock(background, callback)
}
