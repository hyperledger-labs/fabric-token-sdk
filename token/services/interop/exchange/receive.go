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

const receiveTimeout = 240 * time.Second

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
			return nil, errors.Errorf("Received error message %s; with session info %v", string(msg.Payload), context.Session().Info())
		}
		tx, err := newTransactionFromBytes(context, f.network, f.channel, msg.Payload)
		if err != nil {
			return nil, err
		}
		return tx, nil
	case <-time.After(receiveTimeout):
		return nil, errors.Errorf("Timeout reached; with session info %v", context.Session().Info())
	}
}

func newTransactionFromBytes(ctx view.Context, network, channel string, raw []byte) (*Transaction, error) {
	tx, err := ttx.NewTransactionFromBytes(ctx, network, channel, raw)
	if err != nil {
		return nil, err
	}
	return &Transaction{
		Transaction: tx,
	}, nil
}

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
