/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package ttxcc

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
)

type orderingView struct {
	tx *Transaction
}

func NewOrderingView(tx *Transaction) *orderingView {
	return &orderingView{tx: tx}
}

func (o *orderingView) Call(context view.Context) (interface{}, error) {
	if err := fabric.GetDefaultNetwork(context).Ordering().Broadcast(o.tx.Payload.FabricEnvelope); err != nil {
		return nil, err
	}
	return nil, fabric.GetChannel(context, o.tx.Network(), o.tx.Channel()).Finality().IsFinal(o.tx.ID())
}
