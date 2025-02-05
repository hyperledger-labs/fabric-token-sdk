/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package lookup

import (
	"context"
	"encoding/json"

	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/finality"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/tcc"
	"github.com/pkg/errors"
)

const (
	QueryStates = tcc.QueryStates
)

type DeliveryScanQueryByID struct {
	Channel *fabric.Channel
}

func (q *DeliveryScanQueryByID) QueryByID(ctx context.Context, lastBlock driver2.BlockNum, evicted map[driver2.TxID][]finality.ListenerEntry[TxInfo]) (<-chan []TxInfo, error) {
	// collects keys by namespace
	keysByNS := map[driver2.Namespace][]string{}
	for k, v := range evicted {
		ns := v[0].Namespace()
		_, ok := keysByNS[ns]
		if !ok {
			keysByNS[ns] = []string{}
		}
		keysByNS[ns] = append(keysByNS[ns], k)
	}

	ch := make(chan []TxInfo, len(evicted))
	// for each namespace, have a call to the token chaincode
	for ns, keys := range keysByNS {
		arg, err := json.Marshal(keys)
		if err != nil {
			return nil, errors.Wrapf(err, "failed marshalling args for query by ids [%v]", keys)
		}

		logger.Debugf("querying chaincode [%s] for the states of ids [%v]", ns, keys)
		chaincode := q.Channel.Chaincode(ns)
		res, err := chaincode.Query(QueryStates, arg).Query()
		if err != nil {
			return nil, errors.Wrapf(err, "failed querying by ids [%v]", keys)
		}
		values := make([][]byte, 0, len(keys))
		err = json.Unmarshal(res, &values)
		if err != nil {
			return nil, errors.Wrapf(err, "failed unmarshalling results for query by ids [%v]", keys)
		}
		infos := make([]TxInfo, 0, len(values))
		for i, value := range values {
			infos = append(infos, TxInfo{
				Namespace: ns,
				Key:       keys[i],
				Value:     value,
			})
		}
		ch <- infos
	}

	return ch, nil
}
