/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package ttx

import (
	"context"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
)

type finalityView struct {
	tx        *Transaction
	endpoints []view.Identity
	timeout   time.Duration
}

// NewFinalityView returns an instance of the finalityView.
// The view does the following: It waits for the finality of the passed transaction.
// If the transaction is final, the vault is updated.
func NewFinalityView(tx *Transaction) *finalityView {
	return &finalityView{tx: tx}
}

// NewFinalityWithTimeoutView returns an instance of the finalityView.
// The view does the following: It waits for the finality of the passed transaction.
// If the transaction is final, the vault is updated.
// It returns in case the operation is not completed before the passed timeout.
func NewFinalityWithTimeoutView(tx *Transaction, timeout time.Duration) *finalityView {
	return &finalityView{tx: tx, timeout: timeout}
}

// Call executes the view.
// The view does the following: It waits for the finality of the passed transaction.
// If the transaction is final, the vault is updated.
func (f *finalityView) Call(ctx view.Context) (interface{}, error) {
	if len(f.endpoints) != 0 {
		return nil, network.GetInstance(ctx, f.tx.Network(), f.tx.Channel()).IsFinalForParties(f.tx.ID(), f.endpoints...)
	}

	c := ctx.Context()
	if f.timeout != 0 {
		var cancel context.CancelFunc
		c, cancel = context.WithTimeout(c, f.timeout)
		defer cancel()
	}
	return nil, network.GetInstance(ctx, f.tx.Network(), f.tx.Channel()).IsFinal(c, f.tx.ID())
}
