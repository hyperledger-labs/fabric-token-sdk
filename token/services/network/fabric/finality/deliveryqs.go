/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package finality

import (
	"context"
	"fmt"
	"strings"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/finality"
)

type DeliveryScanQueryByID struct {
	Delivery *fabric.Delivery
	Ledger   *fabric.Ledger
	Mapper   finality.TxInfoMapper[TxInfo]
}

func (q *DeliveryScanQueryByID) QueryByID(evicted map[driver.TxID][]finality.ListenerEntry[TxInfo]) (<-chan []TxInfo, error) {
	txIDs := collections.Keys(evicted)
	logger.Debugf("Launching routine to scan for txs [%v]", txIDs)

	results := collections.NewSet(txIDs...)
	ch := make(chan []TxInfo, len(txIDs))

	// for each txID, fetch the corresponding transaction.
	// if the transaction is not found, start a delivery for it
	for _, txID := range txIDs {
		logger.Debugf("loading transaction [%s] from ledger...", txID)
		pt, err := q.Ledger.GetTransactionByID(txID)
		if err == nil {
			logger.Debugf("transaction [%s] found on ledger", txID)
			infos, err := q.Mapper.MapProcessedTx(pt)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to map tx [%s]", txID)
			}
			ch <- infos
			continue
		}

		// which kind of error do we have here?
		if strings.Contains(err.Error(), fmt.Sprintf("TXID [%s] not available", txID)) {
			// transaction was not found
			logger.Errorf("tx [%s] not found on the ledger [%s], start a scan for future transactions", txID, err)
			// start delivery for the future
			// TODO: find a better starting point
			err := q.Delivery.Scan(context.TODO(), "", func(tx *fabric.ProcessedTransaction) (bool, error) {
				if !results.Contains(tx.TxID()) {
					return false, nil
				}

				logger.Debugf("Received result for tx [%s, %v, %d]...", tx.TxID(), tx.ValidationCode(), len(tx.Results()))
				infos, err := q.Mapper.MapProcessedTx(tx)
				if err != nil {
					logger.Errorf("failed mapping tx [%s]: %v", tx.TxID(), err)
					return true, err
				}
				ch <- infos
				results.Remove(tx.TxID())

				return results.Length() == 0, nil
			})
			if err != nil {
				logger.Errorf("Failed scanning: %v", err)
				return nil, err
			}

			continue
		}

		// error not recoverable, fail
		logger.Debugf("scan for tx [%s] failed with err [%s]", txID, err)
		return nil, errors.Wrapf(err, "failed scanning tx [%s]", txID)
	}

	return ch, nil
}
