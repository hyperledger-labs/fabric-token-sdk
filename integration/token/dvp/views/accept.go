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
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type AcceptCashView struct{}

func (a *AcceptCashView) Call(context view.Context) (interface{}, error) {
	// Respond to a request for an identity
	id, err := ttx.RespondRequestRecipientIdentity(context)
	assert.NoError(err, "failed to respond to identity request")

	// Expect a transaction
	tx, err := ttx.ReceiveTransaction(context)
	assert.NoError(err, "failed to receive tokens")

	// Check that the transaction is as expected
	outputs, err := tx.Outputs()
	assert.NoError(err, "failed getting outputs")
	assert.True(outputs.Count() > 0)
	assert.True(outputs.ByRecipient(id).Count() > 0)

	unpsentTokens, err := ttx.MyWallet(context).ListUnspentTokens(ttx.WithType(outputs.At(0).Type))
	assert.NoError(err, "failed retrieving the unspent tokens for type [%s]", outputs.At(0).Type)
	assert.True(unpsentTokens.Sum(64).Cmp(token2.NewQuantityFromUInt64(220)) <= 0, "cannot have more than 220 unspent quantity for type [%s]", outputs.At(0).Type)

	// Accept and send back
	_, err = context.RunView(ttx.NewAcceptView(tx))
	assert.NoError(err, "failed to accept new tokens")

	// Wait for finality
	_, err = context.RunView(ttx.NewFinalityView(tx))
	assert.NoError(err, "new tokens were not committed")

	return nil, nil
}

type AcceptHouseView struct{}

func (a *AcceptHouseView) Call(context view.Context) (interface{}, error) {
	// Respond to a request for an identity
	_, err := state.RespondRequestRecipientIdentity(context)
	assert.NoError(err, "failed to respond to identity request")

	tx, err := state.ReceiveTransaction(context)
	assert.NoError(err)

	// TODO: Check tx

	// Send it back endorsed
	_, err = context.RunView(state.NewAcceptView(tx))
	assert.NoError(err)

	// Wait for confirmation
	return context.RunView(state.NewFinalityView(tx))
}
