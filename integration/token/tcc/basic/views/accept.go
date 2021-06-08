/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package views

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxcc"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type AcceptCashView struct{}

func (a *AcceptCashView) Call(context view.Context) (interface{}, error) {
	// Respond to a request for an identity
	id, err := ttxcc.RespondRequestRecipientIdentity(context)
	assert.NoError(err, "failed to respond to identity request")

	// Expect a transaction
	tx, err := ttxcc.ReceiveTransaction(context)
	assert.NoError(err, "failed to receive tokens")

	// Check that the transaction is as expected
	outputs, err := tx.Outputs()
	assert.NoError(err, "failed getting outputs")
	assert.True(outputs.Count() > 0)
	assert.True(outputs.ByRecipient(id).Count() > 0)

	unpsentTokens, err := ttxcc.MyWallet(context).ListTokens(ttxcc.WithType(outputs.At(0).Type))
	assert.NoError(err, "failed retrieving the unspent tokens for type [%s]", outputs.At(0).Type)
	assert.True(unpsentTokens.Sum(64).Cmp(token2.NewQuantityFromUInt64(3000)) <= 0, "cannot have more than 3000 unspent quantity for type [%s]", outputs.At(0).Type)

	// Accept and send back
	_, err = context.RunView(ttxcc.NewAcceptView(tx))
	assert.NoError(err, "failed to accept new tokens")

	// Wait for finality
	_, err = context.RunView(ttxcc.NewFinalityView(tx))
	assert.NoError(err, "new tokens were not committed")

	return nil, nil
}
