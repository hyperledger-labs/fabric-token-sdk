/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package exchange

import (
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/pkg/errors"
)

func NewCollectEndorsementsView(tx *Transaction) view.View {
	return ttx.NewCollectEndorsementsView(tx.Transaction)
}

type receiveTransactionView struct {
	network string
	channel string
}

func NewReceiveTransactionView(network string) *receiveTransactionView {
	return &receiveTransactionView{network: network}
}

func (f *receiveTransactionView) Call(context view.Context) (interface{}, error) {
	// Wait to receive a transaction back
	ch := context.Session().Receive()

	select {
	case msg := <-ch:
		if msg.Status == view.ERROR {
			return nil, errors.New(string(msg.Payload))
		}
		tx, err := NewTransactionFromBytes(context, f.network, f.channel, msg.Payload)
		if err != nil {
			return nil, err
		}
		return tx, nil
	case <-time.After(240 * time.Second):
		return nil, errors.New("timeout reached")
	}
}
