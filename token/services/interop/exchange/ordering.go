/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package exchange

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
)

func NewOrderingAndFinalityView(tx *Transaction) view.View {
	return ttx.NewOrderingAndFinalityView(tx.Transaction)
}
