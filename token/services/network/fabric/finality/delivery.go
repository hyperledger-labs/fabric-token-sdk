/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package finality

import (
	"context"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
)

type Delivery struct {
	*fabric.Delivery
	*fabric.Ledger
	Logger logging.Logger
}

func (d *Delivery) ScanBlock(background context.Context, callback fabric.BlockCallback) error {
	startingBlock := uint64(0)
	if d.Ledger != nil {
		info, err := d.Ledger.GetLedgerInfo()
		if err == nil {
			startingBlock = info.Height
		} else {
			d.Logger.Errorf("failed to get ledger info: %s", err)
		}
	}
	return d.Delivery.ScanBlockFrom(background, startingBlock, callback)
}
