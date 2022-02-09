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
	// The recipient of a token (issued or transfer) responds, as first operation,
	// to a request for a recipient.
	// The recipient can do that by using the following code.
	// The recipient identity will be taken from the default wallet (ttxcc.MyWallet(context)), if not otherwise specified.
	id, err := ttxcc.RespondRequestRecipientIdentity(context)
	assert.NoError(err, "failed to respond to identity request")

	// At some point, the recipient receives the token transaction that in the mean time has been assembled
	tx, err := ttxcc.ReceiveTransaction(context)
	assert.NoError(err, "failed to receive tokens")

	// The recipient can perform any check on the transaction as required by the business process
	// In particular, here, the recipient checks that the transaction contains at least one output, and
	// that there is at least one output that names the recipient. (The recipient is receiving something.
	outputs, err := tx.Outputs()
	assert.NoError(err, "failed getting outputs")
	assert.True(outputs.Count() > 0)
	assert.True(outputs.ByRecipient(id).Count() > 0)

	// The recipient here is checking that, for each type of token she is receiving,
	// she does not hold already more than 3000 units of that type.
	// Just a fancy query to show the capabilities of the services we are using.
	for _, output := range outputs.ByRecipient(id).Outputs() {
		unspentTokens, err := ttxcc.MyWallet(context).ListUnspentTokens(ttxcc.WithType(output.Type))
		assert.NoError(err, "failed retrieving the unspent tokens for type [%s]", output.Type)
		assert.True(
			unspentTokens.Sum(64).Cmp(token2.NewQuantityFromUInt64(3000)) <= 0,
			"cannot have more than 3000 unspent quantity for type [%s]", output.Type,
		)
	}

	// If everything is fine, the recipient accepts and sends back her signature.
	// Notice that, a signature from the recipient might or might not be required to make the transaction valid.
	// This depends on the driver implementation.
	_, err = context.RunView(ttxcc.NewAcceptView(tx))
	assert.NoError(err, "failed to accept new tokens")

	// Before completing, the recipient waits for finality of the transaction
	_, err = context.RunView(ttxcc.NewFinalityView(tx))
	assert.NoError(err, "new tokens were not committed")

	return nil, nil
}
