/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package views

import (
	"fmt"
	"math/big"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type AuditView struct{}

func (a *AuditView) Call(context view.Context) (interface{}, error) {
	tx, err := ttx.ReceiveTransaction(context)
	assert.NoError(err, "failed receiving transaction")

	w := ttx.MyAuditorWallet(context)
	assert.NotNil(w, "failed getting default auditor wallet")

	// Validate
	auditor := ttx.NewAuditor(context, w)
	assert.NoError(auditor.Validate(tx), "failed auditing verification")

	// Check Metadata
	opRaw := tx.ApplicationMetadata("github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/issue")
	if len(opRaw) != 0 {
		assert.Equal([]byte("issue"), opRaw, "expected 'issue' application metadata")
		metaRaw := tx.ApplicationMetadata("github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/meta")
		assert.Equal([]byte("meta"), metaRaw, "expected 'meta' application metadata")
	}

	// Check limits

	inputs, outputs, err := auditor.Audit(tx)
	assert.NoError(err, "failed retrieving inputs and outputs")

	aqe := auditor.NewQueryExecutor()
	defer aqe.Done()

	// R1: Default payment limit is set to 200. All payments of an amount less than or equal to Default Payment Limit is valid.
	eIDs := inputs.EnrollmentIDs()
	tokenTypes := inputs.TokenTypes()
	fmt.Printf("Limits on inputs [%v][%v]\n", eIDs, tokenTypes)
	for _, eID := range eIDs {
		assert.NotEmpty(eID, "enrollment id should not be empty")
		for _, tokenType := range tokenTypes {
			// compute the payment done in the transaction
			sent := inputs.ByEnrollmentID(eID).ByType(tokenType).Sum().ToBigInt()
			received := outputs.ByEnrollmentID(eID).ByType(tokenType).Sum().ToBigInt()
			fmt.Printf("Payment Limit: [%s] Sent [%d], Received [%d], type [%s]\n", eID, sent.Int64(), received.Int64(), tokenType)

			diff := big.NewInt(0).Sub(sent, received)
			if diff.Cmp(big.NewInt(0)) <= 0 {
				continue
			}
			fmt.Printf("Payment Limit: [%s] Diff [%d], type [%s]\n", eID, diff.Int64(), tokenType)

			assert.True(diff.Cmp(big.NewInt(200)) <= 0, "payment limit reached [%s][%s][%s]", eID, tokenType, diff.Text(10))
			// R3: The default configuration is customized by a specific organisation (Guarantor)
		}
	}

	// R2: Default cumulative payment limit is set to 2000.
	for _, eID := range eIDs {
		assert.NotEmpty(eID, "enrollment id should not be empty")
		for _, tokenType := range tokenTypes {
			// compute the payment done in the transaction
			sent := inputs.ByEnrollmentID(eID).ByType(tokenType).Sum().ToBigInt()
			received := outputs.ByEnrollmentID(eID).ByType(tokenType).Sum().ToBigInt()
			fmt.Printf("Cumulative Limit: [%s] Sent [%d], Received [%d], type [%s]\n", eID, sent.Int64(), received.Int64(), tokenType)

			diff := sent.Sub(sent, received)
			if diff.Cmp(big.NewInt(0)) <= 0 {
				continue
			}
			fmt.Printf("Cumulative Limit: [%s] Diff [%d], type [%s]\n", eID, diff.Int64(), tokenType)

			// load last 10 payments, add diff, and check that it is below the threshold
			filter, err := aqe.NewPaymentsFilter().ByEnrollmentId(eID).ByType(tokenType).Last(10).Execute()
			assert.NoError(err, "failed retrieving last 10 payments")
			sumLastPayments := filter.Sum()
			fmt.Printf("Cumulative Limit: [%s] Last NewPaymentsFilter [%s], type [%s]\n", eID, sumLastPayments.Decimal(), tokenType)

			// R3: The default configuration is customized by a specific organisation (Guarantor)
			total := sumLastPayments.Add(token2.NewQuantityFromBig64(diff))
			assert.True(total.Cmp(token2.NewQuantityFromUInt64(2000)) < 0, "cumulative payment limit reached [%s][%s][%s]", eID, tokenType, total.Decimal())
		}
	}

	// R4: Default holding limit is set to 3000.
	eIDs = outputs.EnrollmentIDs()
	tokenTypes = outputs.TokenTypes()
	for _, eID := range eIDs {
		assert.NotEmpty(eID, "enrollment id should not be empty")
		for _, tokenType := range tokenTypes {
			// compute the amount received
			received := outputs.ByEnrollmentID(eID).ByType(tokenType).Sum().ToBigInt()
			sent := inputs.ByEnrollmentID(eID).ByType(tokenType).Sum().ToBigInt()
			fmt.Printf("Holding Limit: [%s] Sent [%d], Received [%d], type [%s]\n", eID, sent.Int64(), received.Int64(), tokenType)

			diff := received.Sub(received, sent)
			if diff.Cmp(big.NewInt(0)) <= 0 {
				// Nothing received
				continue
			}
			fmt.Printf("Holding Limit: [%s] Diff [%d], type [%s]\n", eID, diff.Int64(), tokenType)

			// load current holding, add diff, and check that it is below the threshold
			filter, err := aqe.NewHoldingsFilter().ByEnrollmentId(eID).ByType(tokenType).Execute()
			assert.NoError(err, "failed retrieving holding for [%s][%s]", eIDs, tokenTypes)
			currentHolding := filter.Sum()

			fmt.Printf("Holding Limit: [%s] Current [%s], type [%s]\n", eID, currentHolding.Decimal(), tokenType)

			total := currentHolding.Add(token2.NewQuantityFromBig64(diff))
			assert.True(total.Cmp(token2.NewQuantityFromUInt64(3000)) < 0, "holding limit reached [%s][%s][%s]", eID, tokenType, total.Decimal())
		}
	}
	aqe.Done()

	return context.RunView(ttx.NewAuditApproveView(w, tx))
}

type RegisterAuditorView struct{}

func (r *RegisterAuditorView) Call(context view.Context) (interface{}, error) {
	return context.RunView(ttx.NewRegisterAuditorView(
		&AuditView{},
	))
}

type RegisterAuditorViewFactory struct{}

func (p *RegisterAuditorViewFactory) NewView(in []byte) (view.View, error) {
	f := &RegisterAuditorView{}
	return f, nil
}
