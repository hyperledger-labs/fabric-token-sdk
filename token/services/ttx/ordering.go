/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/pkg/errors"
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
	options, err := CompileOpts(o.opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile options")
	}
	if err := o.broadcast(context, options.Transaction); err != nil {
		return nil, err
	}

	// cache the token request into the tokens db
	t, err := tokens.GetService(context, options.Transaction.TMSID())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get tokens db for [%s]", options.Transaction.TMSID())
	}
	if !options.NoCachingRequest {
		if err := t.CacheRequest(context.Context(), options.Transaction.TMSID(), options.Transaction.TokenRequest); err != nil {
			logger.Warnf("failed to cache token request [%s], this might cause delay, investigate when possible: [%s]", options.Transaction.TokenRequest.Anchor, err)
		}
	}
	return nil, nil
}

func (o *orderingView) broadcast(context view.Context, transaction *Transaction) error {
	if transaction == nil {
		return errors.Errorf("transaction is nil")
	}
	nw := network.GetInstance(context, transaction.Network(), transaction.Channel())
	if nw == nil {
		return errors.Errorf("network [%s] not found", transaction.Network())
	}
	if err := nw.Broadcast(context.Context(), transaction.Envelope); err != nil {
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
