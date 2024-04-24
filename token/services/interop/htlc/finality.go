/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package htlc

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
)

// NewFinalityView returns an instance of the ttx FinalityView
func NewFinalityView(tx *Transaction, opts ...ttx.TxOption) view.View {
	return ttx.NewFinalityView(tx.Transaction, opts...)
}
