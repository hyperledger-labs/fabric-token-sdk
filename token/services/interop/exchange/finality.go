/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package exchange

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
)

// NewFinalityView returns an instance of the finalityView.
// The view does the following: It waits for the finality of the passed transaction.
// If the transaction is final, the vault is updated.
func NewFinalityView(tx *Transaction) view.View {
	return ttx.NewFinalityView(tx.Transaction)
}
