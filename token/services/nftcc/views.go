/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nftcc

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxcc"
)

func NewCollectEndorsementsView(tx *Transaction) view.View {
	return ttxcc.NewCollectEndorsementsView(tx.Transaction)
}

func NewOrderingAndFinalityView(tx *Transaction) view.View {
	return ttxcc.NewOrderingAndFinalityView(tx.Transaction)
}

func NewFinalityView(tx *Transaction) view.View {
	return ttxcc.NewFinalityView(tx.Transaction)
}

func NewAcceptView(tx *Transaction) view.View {
	return ttxcc.NewAcceptView(tx.Transaction)
}
