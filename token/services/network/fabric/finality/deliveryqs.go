/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package finality

import (
	"context"
	"fmt"
	"strings"

	errors2 "github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	events2 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/events"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/finality"
)

const (
	NumberPastBlocks = 10
	FirstBlock       = 1
)

type DeliveryScanQueryByID struct {
	Delivery *fabric.Delivery
	Ledger   *fabric.Ledger
	Mapper   events2.EventInfoMapper[TxInfo]
}

func (q *DeliveryScanQueryByID) QueryByID(ctx context.Context, lastBlock driver.BlockNum, evicted map[driver.TxID][]events2.ListenerEntry[TxInfo]) (<-chan []TxInfo, error) {
	keys := collections.Keys(evicted)
	ch := make(chan []TxInfo, len(keys))
	go q.queryByID(ctx, keys, ch, lastBlock)
	return ch, nil
}

func (q *DeliveryScanQueryByID) queryByID(ctx context.Context, keys []driver.TxID, ch chan []TxInfo, lastBlock uint64) {
	defer close(ch)

	keySet := collections.NewSet(keys...)

	// for each txID, fetch the corresponding transaction.
	// if the transaction is not found, start a delivery for it
	startDelivery := false
	for _, txID := range keySet.ToSlice() {
		logger.DebugfContext(ctx, "loading transaction [%s] from ledger...", txID)
		pt, err := q.Ledger.GetTransactionByID(txID)
		if err == nil {
			logger.DebugfContext(ctx, "transaction [%s] found on ledger", txID)
			infos, err := q.Mapper.MapProcessedTx(pt)
			if err != nil {
				logger.Errorf("failed to map tx [%s]: [%s]", txID, err)
				return
			}
			keySet.Remove(txID)
			ch <- infos
			continue
		}

		// which kind of error do we have here?
		// TODO: AF In FSC, we have to map the error from Ledger.GetTransactionByID to TxNotFound instead of using substrings
		if strings.Contains(err.Error(), fmt.Sprintf("TXID [%s] not available", txID)) ||
			strings.Contains(err.Error(), fmt.Sprintf("no such transaction ID [%s]", txID)) ||
			errors2.HasType(err, finality.TxNotFound) {
			// transaction was not found
			logger.Errorf("tx [%s] not found on the ledger [%s]", txID, err)
			startDelivery = true
			continue
		}

		// error not recoverable, fail
		logger.DebugfContext(ctx, "scan for tx [%s] failed with err [%s]", txID, err)
		return
	}

	if !startDelivery {
		return
	}

	startingBlock := max(FirstBlock, lastBlock-NumberPastBlocks)
	// startingBlock := uint64(0)
	logger.DebugfContext(ctx, "start scanning blocks starting from [%d], looking for remaining keys [%s]", startingBlock, keySet)

	// start delivery for the future
	err := q.Delivery.ScanFromBlock(
		ctx,
		startingBlock,
		func(tx *fabric.ProcessedTransaction) (bool, error) {
			if !keySet.Contains(tx.TxID()) {
				return false, nil
			}
			logger.DebugfContext(ctx, "received result for tx [%s, %v, %d]...", tx.TxID(), tx.ValidationCode(), len(tx.Results()))
			infos, err := q.Mapper.MapProcessedTx(tx)
			if err != nil {
				logger.Errorf("failed mapping tx [%s]: %v", tx.TxID(), err)
				return true, err
			}
			ch <- infos
			keySet.Remove(tx.TxID())
			logger.DebugfContext(ctx, "removing [%s] from searching list, remaining keys [%d]", tx.TxID(), keySet.Length())
			return keySet.Length() == 0, nil
		},
	)
	if err != nil {
		logger.Errorf("failed scanning blocks [%s], started from [%d]", err, startingBlock)
		return
	}
	logger.DebugfContext(ctx, "finished scanning blocks starting from [%d]", startingBlock)
}
