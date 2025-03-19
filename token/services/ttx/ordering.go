/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"context"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/pkg/errors"
)

type orderingView struct {
	opts []TxOption
}

// NewOrderingView returns a new instance of the OrderingView struct.
// The view does the following:
// 1. It broadcasts the token transaction to the proper backend.
func NewOrderingView(tx *Transaction, opts ...TxOption) *orderingView {
	return NewOrderingViewWithOpts(append([]TxOption{WithTransactions(tx)}, opts...)...)
}

func NewOrderingViewWithOpts(opts ...TxOption) *orderingView {
	return &orderingView{opts: opts}
}

func (o *orderingView) Call(context view.Context) (interface{}, error) {
	return context.RunView(&OrderingView{
		opts:            o.opts,
		tokensManager:   utils.MustGet(context.GetService(&tokens.Manager{})).(*tokens.Manager),
		networkProvider: network.GetProvider(context),
	})
}

type OrderingView struct {
	opts []TxOption

	tokensManager   *tokens.Manager
	networkProvider *network.Provider
}

// Call execute the view.
// The view does the following:
// 1. It broadcasts the token transaction to the proper backend.
func (o *OrderingView) Call(context view.Context) (interface{}, error) {
	// Compile options
	options, err := CompileOpts(o.opts...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compile options")
	}
	if err := o.broadcast(context.Context(), options.Transaction); err != nil {
		return nil, err
	}

	// cache the token request into the tokens db
	t, err := o.tokensManager.Tokens(options.Transaction.TMSID())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get tokens db for [%s]", options.Transaction.TMSID())
	}
	if !options.NoCachingRequest {
		if err := t.CacheRequest(options.Transaction.TMSID(), options.Transaction.TokenRequest); err != nil {
			logger.Warnf("failed to cache token request [%s], this might cause delay, investigate when possible: [%s]", options.Transaction.TokenRequest.Anchor, err)
		}
	}
	return nil, nil
}

func (o *OrderingView) broadcast(context context.Context, transaction *Transaction) error {
	if transaction == nil {
		return errors.Errorf("transaction is nil")
	}
	nw, err := o.networkProvider.GetNetwork(transaction.Network(), transaction.Channel())
	if err != nil {
		return errors.Errorf("network [%s] not found", transaction.Network())
	}
	if err := nw.Broadcast(context, transaction.Payload.Envelope); err != nil {
		return errors.WithMessagef(err, "failed to broadcast token transaction [%s]", transaction.ID())
	}
	return nil
}

type OrderingViewFactory struct {
	tokensManager   *tokens.Manager
	networkProvider *network.Provider
}

func NewOrderingViewFactory(
	tokensManager *tokens.Manager,
	networkProvider *network.Provider,
) *OrderingViewFactory {
	return &OrderingViewFactory{
		tokensManager:   tokensManager,
		networkProvider: networkProvider,
	}
}

func (f *OrderingViewFactory) New(opts ...TxOption) (*OrderingView, error) {
	return &OrderingView{
		opts:            opts,
		tokensManager:   f.tokensManager,
		networkProvider: f.networkProvider,
	}, nil
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

func NewOrderingAndFinalityViewFactory(
	finalityView FinalityViewFactory,
	orderingView OrderingViewFactory,
) *OrderingAndFinalityViewFactory {
	return &OrderingAndFinalityViewFactory{
		finalityView: finalityView,
		orderingView: orderingView,
	}
}

func (f *OrderingAndFinalityViewFactory) New(tx *Transaction, timeout time.Duration) (*OrderingAndFinalityView, error) {
	return &OrderingAndFinalityView{
		tx:           tx,
		timeout:      timeout,
		finalityView: f.finalityView,
		orderingView: f.orderingView,
	}, nil
}

type OrderingAndFinalityViewFactory struct {
	finalityView FinalityViewFactory
	orderingView OrderingViewFactory
}

type OrderingAndFinalityView struct {
	tx      *Transaction
	timeout time.Duration

	finalityView FinalityViewFactory
	orderingView OrderingViewFactory
}

func (o *OrderingAndFinalityView) Call(ctx view.Context) (interface{}, error) {
	if _, err := ctx.RunView(utils.MustGet(o.orderingView.New(WithTransactions(o.tx)))); err != nil {
		return nil, err
	}
	return ctx.RunView(utils.MustGet(o.finalityView.New(1*time.Second, WithTransactions(o.tx), WithTimeout(o.timeout))))
}
