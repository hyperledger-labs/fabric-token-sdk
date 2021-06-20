/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package ttxcc

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
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
	fs := fabric.GetChannel(context, f.tx.Network(), f.tx.Channel()).Finality()
	if len(f.endpoints) != 0 {
		return nil, fs.IsFinalForParties(f.tx.ID(), f.endpoints...)
	}
	return nil, fs.IsFinal(f.tx.ID())
}
