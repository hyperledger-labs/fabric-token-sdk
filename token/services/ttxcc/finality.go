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

func (f *finalityView) Call(context view.Context) (interface{}, error) {
	fs := fabric.GetChannel(context, f.tx.Network(), f.tx.Channel()).Finality()
	if len(f.endpoints) != 0 {
		return nil, fs.IsFinalForParties(f.tx.ID(), f.endpoints...)
	}
	return nil, fs.IsFinal(f.tx.ID())
}

func NewFinalityView(tx *Transaction) *finalityView {
	return &finalityView{tx: tx}
}
