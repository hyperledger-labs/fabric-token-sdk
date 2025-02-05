/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package lookup

import (
	"context"
	"encoding/json"
	"slices"

	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/finality"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/tcc"
	"go.uber.org/zap"
)

const (
	QueryStates = tcc.QueryStates
)

type DeliveryScanQueryByID struct {
	Delivery *fabric.Delivery
	Channel  *fabric.Channel
}

func (q *DeliveryScanQueryByID) QueryByID(ctx context.Context, lastBlock driver2.BlockNum, evicted map[driver2.TxID][]finality.ListenerEntry[TxInfo]) (<-chan []TxInfo, error) {
	// we are abusing TxID to carry the name of the keys we are looking for.
	// Keys are supposed to be unique
	keys := collections.Keys(evicted) // These are the state keys we are looking for
	results := collections.NewSet(keys...)
	ch := make(chan []TxInfo, len(keys))
	go q.queryByID(ctx, results, ch, lastBlock, evicted)
	return ch, nil
}

func (q *DeliveryScanQueryByID) queryByID(ctx context.Context, results collections.Set[string], ch chan []TxInfo, lastBlock uint64, evicted map[driver2.TxID][]finality.ListenerEntry[TxInfo]) {
	defer close(ch)

	// group keys by namespace
	keysByNS := map[driver2.Namespace][]string{}
	for k, v := range evicted {
		ns := v[0].Namespace()
		_, ok := keysByNS[ns]
		if !ok {
			keysByNS[ns] = []string{}
		}
		keysByNS[ns] = append(keysByNS[ns], k)
	}

	// for each namespace, have a call to the token chaincode
	startDelivery := false
	for ns, keys := range keysByNS {
		arg, err := json.Marshal(keys)
		if err != nil {
			logger.Error("failed marshalling args for query by ids [%v]: [%s]", keys, err)
			return
		}

		logger.Debugf("querying chaincode [%s] for the states of ids [%v]", ns, keys)
		chaincode := q.Channel.Chaincode(ns)
		res, err := chaincode.Query(QueryStates, arg).Query()
		if err != nil {
			logger.Errorf("failed querying by ids [%v]: [%s]", keys, err)
			return
		}
		values := make([][]byte, 0, len(keys))
		err = json.Unmarshal(res, &values)
		if err != nil {
			logger.Errorf("failed unmarshalling results for query by ids [%v]: [%s]", keys, err)
			return
		}
		infos := make([]TxInfo, 0, len(values))
		var remainingKeys []string
		for i, value := range values {
			if len(value) == 0 {
				startDelivery = true
				remainingKeys = append(remainingKeys, keys[i])
				continue
			}
			infos = append(infos, TxInfo{
				Namespace: ns,
				Key:       keys[i],
				Value:     value,
			})
			results.Remove(keys[i])
		}
		if len(remainingKeys) == 0 {
			delete(keysByNS, ns)
		} else {
			keysByNS[ns] = remainingKeys
		}
		ch <- infos
	}

	if startDelivery {
		startingBlock := finality.MaxUint64(1, lastBlock-10)
		// startingBlock := uint64(0)
		if logger.IsEnabledFor(zap.DebugLevel) {
			logger.Debugf("start scanning blocks starting from [%d], looking for remaining keys [%v]", startingBlock, results.ToSlice())
		}

		// start delivery for the future
		v := q.Channel.Vault()
		err := q.Delivery.ScanFromBlock(
			ctx,
			startingBlock,
			func(tx *fabric.ProcessedTransaction) (bool, error) {
				rws, err := v.InspectRWSet(ctx, tx.Results())
				if err != nil {
					return false, err
				}

				var txInfos []TxInfo
				for namespace, keys := range keysByNS {
					if !slices.Contains(rws.Namespaces(), namespace) {
						logger.Debugf("scanning [%s] does not contain namespace [%s]", tx.TxID(), namespace)
						continue
					}

					for i := 0; i < rws.NumWrites(namespace); i++ {
						k, v, err := rws.GetWriteAt(namespace, i)
						if err != nil {
							logger.Debugf("scanning [%s]: failed to get key [%s]", tx.TxID(), err)
							return false, err
						}
						if slices.Contains(keys, k) {
							logger.Debugf("scanning [%s]: found key [%s]", tx.TxID(), k)
							txInfos = append(txInfos, TxInfo{
								Namespace: namespace,
								Key:       k,
								Value:     v,
							})
							logger.Debugf("removing [%s] from searching list, remaining keys [%d]", k, results.Length())
							results.Remove(k)
						}
					}
				}
				if len(txInfos) != 0 {
					ch <- txInfos
				}

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
