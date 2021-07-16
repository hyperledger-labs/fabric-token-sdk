/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package ttxcc

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/pkg/errors"
)

type orderingView struct {
	tx *Transaction
}

// NewOrderingView returns a new instance of the orderingView struct.
// The view does the following:
// 1. It broadcasts the token token transaction to the proper Fabric ordering service.
func NewOrderingView(tx *Transaction) *orderingView {
	return &orderingView{tx: tx}
}

// Call execute the view.
// The view does the following:
// 1. It broadcasts the token token transaction to the proper Fabric ordering service.
func (o *orderingView) Call(context view.Context) (interface{}, error) {
	if err := fabric.GetFabricNetworkService(context, o.tx.Network()).Ordering().Broadcast(o.tx.Payload.FabricEnvelope); err != nil {
		return nil, err
	}
	return nil, nil
}

type orderingAndFinalityView struct {
	tx *Transaction
}

// NewOrderingAndFinalityView returns a new instance of the orderingAndFinalityView struct.
// The view does the following:
// 1. It broadcasts the token token transaction to the proper Fabric ordering service.
// 2. It waits for finality of the token transaction by listening to delivery events from one of the
// Fabric peer nodes trusted by the FSC node.
func NewOrderingAndFinalityView(tx *Transaction) *orderingAndFinalityView {
	return &orderingAndFinalityView{tx: tx}
}

// Call execute the view.
// The view does the following:
// 1. It broadcasts the token token transaction to the proper Fabric ordering service.
// 2. It waits for finality of the token transaction by listening to delivery events from one of the
// Fabric peer nodes trusted by the FSC node.
func (o *orderingAndFinalityView) Call(context view.Context) (interface{}, error) {
	fns := fabric.GetFabricNetworkService(context, o.tx.Network())
	if fns == nil {
		return nil, errors.Errorf("network [%s] not found", o.tx.Network())
	}
	ch, err := fns.Channel(o.tx.Channel())
	if err != nil {
		return nil, errors.Errorf("channel [%s:%s] not found", o.tx.Network(), o.tx.Channel())
	}
	if err := fns.Ordering().Broadcast(o.tx.Payload.FabricEnvelope); err != nil {
		return nil, err
	}

	return nil, ch.Finality().IsFinal(o.tx.ID())
}
