/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package htlc

import (
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/pkg/errors"
)

// NewCollectEndorsementsView returns an instance of the ttx collectEndorsementsView struct
func NewCollectEndorsementsView(tx *Transaction) view.View {
	return ttx.NewCollectEndorsementsView(tx.Transaction)
}

type receiveTransactionView struct {
	network string
	channel string
}

// NewReceiveTransactionView returns an instance of receiveTransactionView struct
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

// ReceiveTransaction executes the receiveTransactionView and returns the received transaction
func ReceiveTransaction(context view.Context) (*Transaction, error) {
	logger.Debugf("receive a new transaction...")

	txBoxed, err := context.RunView(NewReceiveTransactionView(""))
	if err != nil {
		return nil, err
	}

	cctx, ok := txBoxed.(*Transaction)
	if !ok {
		return nil, errors.Errorf("received transaction of wrong type [%T]", cctx)
	}
	logger.Debugf("received transaction with id [%s]", cctx.ID())

	return cctx, nil
}
