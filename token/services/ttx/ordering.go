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
	tx *Transaction
}

// NewOrderingView returns a new instance of the orderingView struct.
// The view does the following:
// 1. It broadcasts the token transaction to the proper backend.
func NewOrderingView(tx *Transaction) *orderingView {
	return &orderingView{tx: tx}
}

// Call execute the view.
// The view does the following:
// 1. It broadcasts the token transaction to the proper backend.
func (o *orderingView) Call(context view.Context) (interface{}, error) {
	if o.tx.Payload.Envelope == nil {
		return nil, errors.Errorf("envelope is nil for token transaction [%s]", o.tx.ID())
	}

	if len(o.tx.Payload.Envelope.TxID()) == 0 {
		return nil, errors.Errorf("txID is empty for token transaction [%s]", o.tx.ID())
	}

	if err := network.GetInstance(context, o.tx.Network(), o.tx.Channel()).Broadcast(context.Context(), o.tx.Payload.Envelope); err != nil {
		return nil, errors.WithMessagef(err, "failed to broadcast token transaction [%s]", o.tx.ID())
	}
	return nil, nil
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
	return &orderingAndFinalityView{tx: tx}
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
	nw := network.GetInstance(ctx, o.tx.Network(), o.tx.Channel())
	if nw == nil {
		return nil, errors.Errorf("network [%s] not found", o.tx.Network())
	}
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("[%s] broadcasting token transaction [%s]", o.tx.Channel(), o.tx.ID())
	}

	if o.tx.Payload.Envelope == nil {
		return nil, errors.Errorf("envelope is nil for token transaction [%s]", o.tx.ID())
	}

	if len(o.tx.Payload.Envelope.TxID()) == 0 {
		return nil, errors.Errorf("txID is empty for token transaction [%s]", o.tx.ID())
	}

	env := o.tx.Payload.Envelope

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		rawEnv, err := env.Bytes()
		if err != nil {
			return nil, errors.WithMessagef(err, "failed to marshal envelope for token transaction [%s]", o.tx.ID())
		}
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("send for ordering, ttx size [%d], rws [%d], creator [%d]", len(rawEnv), len(env.Results()), len(env.Creator()))
		}
	}

	if err := nw.Broadcast(ctx.Context(), env); err != nil {
		return nil, errors.WithMessagef(err, "failed to broadcast token transaction [%s]", o.tx.ID())
	}

	return ctx.RunView(NewFinalityWithTimeoutView(o.tx, o.timeout))
}
