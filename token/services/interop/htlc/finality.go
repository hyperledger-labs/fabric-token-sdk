/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package htlc

import (
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
)

// NewFinalityView returns an instance of the ttx FinalityView
func NewFinalityView(tx *Transaction) view.View {
	return ttx.NewFinalityView(tx.Transaction)
}

// NewFinalityWithTimeoutView returns an instance of the ttx FinalityView with timeout
func NewFinalityWithTimeoutView(tx *Transaction, timeout time.Duration) view.View {
	return ttx.NewFinalityWithTimeoutView(tx.Transaction, timeout)
}
