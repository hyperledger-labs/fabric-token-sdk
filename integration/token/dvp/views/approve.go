/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package views

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/services/state"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
)

type TokenApproveView struct{}

func (a *TokenApproveView) Call(context view.Context) (interface{}, error) {
	tx, err := ttx.ReceiveTransaction(context)
	assert.NoError(err)

	// Validate transaction
	assert.NoError(ttx.NewApprover(context).Validate(tx))

	// Sign and send back
	_, err = context.RunView(ttx.NewApproveView(tx))
	assert.NoError(err)

	// Wait for confirmation
	return context.RunView(ttx.NewFinalityView(tx))
}

type HouseApproveView struct{}

func (a *HouseApproveView) Call(context view.Context) (interface{}, error) {
	tx, err := state.ReceiveTransaction(context)
	assert.NoError(err)

	// TODO: Validate transaction

	// Sign and send back
	_, err = context.RunView(state.NewEndorseView(tx))
	assert.NoError(err)

	// Wait for confirmation
	return context.RunView(state.NewFinalityView(tx))
}
