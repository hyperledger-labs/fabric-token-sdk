/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

type orderingView struct {
	opts []TxOption
}

// NewOrderingView returns a new instance of the orderingView struct.
// The view does the following:
// 1. It broadcasts the token transaction to the proper backend.
func NewOrderingView(tx *Transaction, opts ...TxOption) *orderingView {
	return NewOrderingViewWithOpts(append([]TxOption{WithTransactions(tx)}, opts...)...)
}

func NewOrderingViewWithOpts(opts ...TxOption) *orderingView {
	return &orderingView{opts: opts}
}

// Call execute the view.
// The view does the following:
// 1. It broadcasts the token transaction to the proper backend.
func (o *orderingView) Call(context view.Context) (interface{}, error) {
	// Compile options
	options, err := compile(o.opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile options")
	}
	if err := o.broadcast(context, options.Transaction); err != nil {
		return nil, err
	}
	return nil, nil
}

func (o *orderingView) broadcast(context view.Context, transaction *Transaction) error {
	if transaction == nil {
		return errors.Errorf("transaction is nil")
	}
	if transaction.Payload.Envelope == nil {
		return errors.Errorf("envelope is nil for token transaction [%s]", transaction.ID())
	}

	if len(transaction.Payload.Envelope.TxID()) == 0 {
		return errors.Errorf("txID is empty for token transaction [%s]", transaction.ID())
	}

	nw := network.GetInstance(context, transaction.Network(), transaction.Channel())
	if nw == nil {
		return errors.Errorf("network [%s] not found", transaction.Network())
	}

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		rawEnv, err := transaction.Payload.Envelope.Bytes()
		if err != nil {
			return errors.WithMessagef(err, "failed to marshal envelope for token transaction [%s]", transaction.ID())
		}
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("send for ordering, ttx size [%d]", len(rawEnv))
		}
	}

	if err := nw.Broadcast(context.Context(), transaction.Payload.Envelope); err != nil {
		return errors.WithMessagef(err, "failed to broadcast token transaction [%s]", transaction.ID())
	}
	return nil
}

type orderingAndFinalityView struct {
	tx      *Transaction
	timeout time.Duration
}

// NewOrderingAndFinalityView returns a new instance of the orderingAndFinalityView struct.
// The view does the following:
// 1. It broadcasts the token transaction to the proper backend.
// 2. It waits for finality of the token transaction by listening to delivery events from one of the
// Fabric peer nodes trusted by the FSC node.
func NewOrderingAndFinalityView(tx *Transaction) *orderingAndFinalityView {
	return NewOrderingAndFinalityWithTimeoutView(tx, finalityTimeout)
}

// NewOrderingAndFinalityWithTimeoutView returns a new instance of the orderingAndFinalityView struct.
// The view does the following:
// 1. It broadcasts the token transaction to the proper backend.
// 2. It waits for finality of the token transaction.
func NewOrderingAndFinalityWithTimeoutView(tx *Transaction, timeout time.Duration) *orderingAndFinalityView {
	return &orderingAndFinalityView{tx: tx, timeout: timeout}
}

// Call executes the view.
// The view does the following:
// 1. It broadcasts the token transaction to the proper backend.
// 2. It waits for finality of the token transaction.
// It returns in case the operation is not completed before the passed timeout.
func (o *orderingAndFinalityView) Call(ctx view.Context) (interface{}, error) {
	if _, err := ctx.RunView(NewOrderingView(o.tx)); err != nil {
		return nil, err
	}
	return ctx.RunView(NewFinalityView(o.tx, WithTimeout(o.timeout)))
}
