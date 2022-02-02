/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package ttxcc

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracker/metrics"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
)

type finalityView struct {
	tx        *Transaction
	endpoints []view.Identity
}

// NewFinalityView returns an instance of the finalityView.
// The view does the following: It waits for the finality of the passed transaction.
// If the transaction is final, the vault is updated.
func NewFinalityView(tx *Transaction) *finalityView {
	return &finalityView{tx: tx}
}

// Call executes the view.
// The view does the following: It waits for the finality of the passed transaction.
// If the transaction is final, the vault is updated.
func (f *finalityView) Call(context view.Context) (interface{}, error) {
	agent := metrics.Get(context)
	agent.EmitKey(0, "ttxcc", "start", "finalityView", f.tx.ID())
	defer agent.EmitKey(0, "ttxcc", "end", "finalityView", f.tx.ID())

	if len(f.endpoints) != 0 {
		return nil, network.GetInstance(context, f.tx.Network(), f.tx.Channel()).IsFinalForParties(f.tx.ID(), f.endpoints...)
	}
	return nil, network.GetInstance(context, f.tx.Network(), f.tx.Channel()).IsFinal(f.tx.ID())
}
