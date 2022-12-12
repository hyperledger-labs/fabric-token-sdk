/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package views

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
)

const (
	Limit int64 = 50
)

type AuditView struct{}

func (a *AuditView) Call(context view.Context) (interface{}, error) {
	tx, err := ttx.ReceiveTransaction(context)
	assert.NoError(err, "failed receiving transaction")

	assert.NoError(tx.IsValid(), "failed verifying transaction")

	w := ttx.MyAuditorWallet(context, token.WithTMSID(tx.TokenService().ID()))
	assert.NotNil(w, "failed getting default auditor wallet")
	auditor := ttx.NewAuditor(context, w)
	assert.NoError(auditor.Validate(tx), "failed auditing verification")

	// Check limits

	// extract inputs and outputs
	inputs, outputs, err := auditor.Audit(tx)
	assert.NoError(err, "failed retrieving inputs and outputs")
	defer auditor.Release(tx)

	// For example, all payments of an amount less than or equal to payment limit is valid
	eIDs := inputs.EnrollmentIDs()
	tokenTypes := inputs.TokenTypes()
	logger.Debugf("Limits on inputs [%v][%v]", eIDs, tokenTypes)
	for _, eID := range eIDs {
		assert.NotEmpty(eID, "enrollment id should not be empty")
		for _, tokenType := range tokenTypes {
			// compute the payment done in the transaction
			sent := inputs.ByEnrollmentID(eID).ByType(tokenType).Sum()
			received := outputs.ByEnrollmentID(eID).ByType(tokenType).Sum()
			logger.Debugf("Payment Limit: [%s] Sent [%d], Received [%d], type [%s]", eID, sent.Int64(), received.Int64(), tokenType)

			diff := big.NewInt(0).Sub(sent, received)
			if diff.Cmp(big.NewInt(0)) <= 0 {
				continue
			}
			logger.Debugf("Payment Limit: [%s] Diff [%d], type [%s]", eID, diff.Int64(), tokenType)

			assert.True(diff.Cmp(big.NewInt(Limit)) <= 0, "payment limit reached [%s][%s][%s]", eID, tokenType, diff.Text(10))
		}
	}

	for i := 0; i < inputs.Count(); i++ {
		input, err := interop.ToInput(inputs.At(i))
		assert.NoError(err, "cannot get htlc input wrapper")
		if input.IsHTLC() {
			// check script details, for example make sure the hash is set
			htlc, err := input.HTLC()
			assert.NoError(err, "cannot get htlc script from input")
			assert.True(len(htlc.HashInfo.Hash) > 0, "hash is not set")
		} else {
			if input.IsPledge() {
				// check that one can retrieve pledge
				_, err := input.Pledge()
				assert.NoError(err, "cannot get pledge script from input")
			}
		}
	}

	now := time.Now()
	for i := 0; i < outputs.Count(); i++ {
		output, err := interop.ToOutput(outputs.At(i))
		assert.NoError(err, "cannot get htlc output wrapper")
		switch {
		case output.IsHTLC():
			// check script details
			htlc, err := output.HTLC()
			assert.NoError(err, "cannot get htlc script from output")
			assert.NoError(htlc.Validate(now), "script is not valid")
		case output.IsPledge():
			pledge, err := output.Pledge()
			assert.NoError(err, "cannot get pledge script from output")
			assert.NoError(pledge.Validate(), "invalid pledge script")
		}
	}

	return context.RunView(ttx.NewAuditApproveView(w, tx))
}

type RegisterAuditor struct {
	TMSID token.TMSID
}

type RegisterAuditorView struct {
	*RegisterAuditor
}

func (r *RegisterAuditorView) Call(context view.Context) (interface{}, error) {
	return context.RunView(ttx.NewRegisterAuditorView(
		&AuditView{},
		token.WithTMSID(r.TMSID),
	))
}

type RegisterAuditorViewFactory struct{}

func (p *RegisterAuditorViewFactory) NewView(in []byte) (view.View, error) {
	f := &RegisterAuditorView{RegisterAuditor: &RegisterAuditor{}}
	err := json.Unmarshal(in, f.RegisterAuditor)
	assert.NoError(err, "failed unmarshalling input")

	return f, nil
}
