/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package finality

import (
	"context"
	"fmt"
	"strings"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/finality"
	"go.uber.org/zap"
)

type DeliveryScanQueryByID struct {
	Delivery *fabric.Delivery
	Ledger   *fabric.Ledger
	Mapper   finality.TxInfoMapper[TxInfo]
}

func (q *DeliveryScanQueryByID) QueryByID(ctx context.Context, lastBlock driver.BlockNum, evicted map[driver.TxID][]finality.ListenerEntry[TxInfo]) (<-chan []TxInfo, error) {
	txIDs := collections.Keys(evicted)
	results := collections.NewSet(txIDs...)
	ch := make(chan []TxInfo, len(txIDs))
	go q.queryByID(ctx, results, ch, lastBlock)
	return ch, nil
}

func (q *DeliveryScanQueryByID) queryByID(ctx context.Context, results collections.Set[string], ch chan []TxInfo, lastBlock uint64) {
	defer close(ch)

	// for each txID, fetch the corresponding transaction.
	// if the transaction is not found, start a delivery for it
	startDelivery := false
	for _, txID := range results.ToSlice() {
		logger.Debugf("loading transaction [%s] from ledger...", txID)
		pt, err := q.Ledger.GetTransactionByID(txID)
		if err == nil {
			logger.Debugf("transaction [%s] found on ledger", txID)
			infos, err := q.Mapper.MapProcessedTx(pt)
			if err != nil {
				logger.Errorf("failed to map tx [%s]: [%s]", txID, err)
				return
			}
			results.Remove(txID)
			ch <- infos
			continue
		}

		// which kind of error do we have here?
		errorMsg := err.Error()
		if strings.Contains(errorMsg, fmt.Sprintf("TXID [%s] not available", txID)) ||
			strings.Contains(errorMsg, fmt.Sprintf("no such transaction ID [%s]", txID)) {
			// transaction was not found
			logger.Errorf("tx [%s] not found on the ledger [%s]", txID, err)
			startDelivery = true
			continue
		}

		// error not recoverable, fail
		logger.Debugf("scan for tx [%s] failed with err [%s]", txID, err)
		return
	}

	if startDelivery {
		startingBlock := MaxUint64(1, lastBlock-10)
		// startingBlock := uint64(0)
		if logger.IsEnabledFor(zap.DebugLevel) {
			logger.Debugf("start scanning blocks starting from [%d], looking for remaining keys [%v]", startingBlock, results.ToSlice())
		}

		// start delivery for the future
		err := q.Delivery.ScanFromBlock(
			ctx,
			startingBlock,
			func(tx *fabric.ProcessedTransaction) (bool, error) {
				if !results.Contains(tx.TxID()) {
					return false, nil
				}
				logger.Debugf("received result for tx [%s, %v, %d]...", tx.TxID(), tx.ValidationCode(), len(tx.Results()))
				infos, err := q.Mapper.MapProcessedTx(tx)
				if err != nil {
					logger.Errorf("failed mapping tx [%s]: %v", tx.TxID(), err)
					return true, err
				}
				ch <- infos
				results.Remove(tx.TxID())
				logger.Debugf("removing [%s] from searching list, remaining keys [%d]", tx.TxID(), results.Length())
				return results.Length() == 0, nil
			},
		)
		if err != nil {
			logger.Errorf("failed scanning blocks [%s], started from [%d]", err, startingBlock)
			return
		}
		logger.Debugf("finished scanning blocks starting from [%d]", startingBlock)
	}
}

func MaxUint64(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}
