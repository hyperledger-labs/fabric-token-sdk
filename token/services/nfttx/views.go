/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nfttx

import (
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
)

func NewCollectEndorsementsView(tx *Transaction) view.View {
	return ttx.NewCollectEndorsementsView(tx.Transaction)
}

func NewOrderingAndFinalityView(tx *Transaction) view.View {
	return ttx.NewOrderingAndFinalityView(tx.Transaction)
}

func NewOrderingAndFinalityWithTimeoutView(tx *Transaction, timeout time.Duration) view.View {
	return ttx.NewOrderingAndFinalityWithTimeoutView(tx.Transaction, timeout)
}

func NewFinalityView(tx *Transaction) view.View {
	return ttx.NewFinalityView(tx.Transaction)
}

func NewFinalityWithTimeoutView(tx *Transaction, timeout time.Duration) view.View {
	return ttx.NewFinalityWithTimeoutView(tx.Transaction, timeout)
}

func NewAcceptView(tx *Transaction) view.View {
	return ttx.NewAcceptView(tx.Transaction)
}
