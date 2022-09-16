/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package views

import (
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/htlc"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
)

type AcceptCashView struct{}

func (a *AcceptCashView) Call(context view.Context) (interface{}, error) {
	// The recipient of a token (issued or transferred) responds, as first operation,
	// to a request for a recipient identity.
	// The recipient can do that by using the following code.
	// The recipient identity will be taken from the default wallet (ttx.MyWallet(context)), if not otherwise specified.
	me, err := ttx.RespondRequestRecipientIdentity(context)
	assert.NoError(err, "failed to respond to identity request")

	// At some point, the recipient receives the token transaction that in the meantime has been assembled
	tx, err := ttx.ReceiveTransaction(context)
	assert.NoError(err, "failed to receive tokens")

	outputs, err := tx.Outputs()
	assert.NoError(err, "failed to get transaction's outputs")
	now := time.Now()
	for i := 0; i < outputs.Count(); i++ {
		output, err := htlc.ToOutput(outputs.At(i))
		assert.NoError(err, "cannot get htlc output wrapper")
		if !output.IsHTLC() {
			continue
		}
		// check script details, for example make sure the deadline has not passed
		script, err := output.Script()
		assert.NoError(err, "cannot get htlc script from input")
		assert.NoError(script.Validate(now), "script is not valid")
		assert.Equal(me, script.Recipient, "recipient does not match [%s]!=[%s]", me, script.Recipient)
	}

	// If everything is fine, the recipient accepts and sends back her signature.
	// Notice that, a signature from the recipient might or might not be required to make the transaction valid.
	// This depends on the driver implementation.
	_, err = context.RunView(ttx.NewAcceptView(tx))
	assert.NoError(err, "failed to accept new tokens")

	// Before completing, the recipient waits for finality of the transaction
	_, err = context.RunView(ttx.NewFinalityView(tx))
	assert.NoError(err, "new tokens were not committed")

	return nil, nil
}
