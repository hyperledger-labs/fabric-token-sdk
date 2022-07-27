/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package views

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type AuditView struct{}

func (a *AuditView) Call(context view.Context) (interface{}, error) {
	logger.Debugf("AuditView: [%s]", context.ID())
	tx, err := ttx.ReceiveTransaction(context)
	assert.NoError(err, "failed receiving transaction")
	logger.Debugf("AuditView: [%s]", tx.ID())

	w := ttx.MyAuditorWallet(context)
	assert.NotNil(w, "failed getting default auditor wallet")

	// Validate
	logger.Debugf("AuditView: get auditor [%s]", tx.ID())
	auditor := ttx.NewAuditor(context, w)
	assert.NoError(auditor.Validate(tx), "failed auditing verification")
	logger.Debugf("AuditView: get auditor done [%s]", tx.ID())

	// Check Metadata
	logger.Debugf("AuditView: check metadata [%s]", tx.ID())
	opRaw := tx.ApplicationMetadata("github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/issue")
	if len(opRaw) != 0 {
		assert.Equal([]byte("issue"), opRaw, "expected 'issue' application metadata")
		metaRaw := tx.ApplicationMetadata("github.com/hyperledger-labs/fabric-token-sdk/integration/token/fungible/meta")
		assert.Equal([]byte("meta"), metaRaw, "expected 'meta' application metadata")
	}
	logger.Debugf("AuditView: check metadata done [%s]", tx.ID())

	// Check limits

	// extract inputs and outputs
	logger.Debugf("AuditView: audit [%s]", tx.ID())
	inputs, outputs, err := auditor.Audit(tx)
	assert.NoError(err, "failed retrieving inputs and outputs")
	logger.Debugf("AuditView: audit done [%s]", tx.ID())

	// acquire locks on inputs and outputs' enrollment IDs
	logger.Debugf("AuditView: acquire locks [%s]", tx.ID())
	assert.NoError(auditor.AcquireLocks(append(inputs.EnrollmentIDs(), outputs.EnrollmentIDs()...)...), "failed acquiring locks")
	defer auditor.Unlock(append(inputs.EnrollmentIDs(), outputs.EnrollmentIDs()...))
	logger.Debugf("AuditView: acquire locks done [%s]", tx.ID())

	logger.Debugf("AuditView: [%s] get query executor... ", tx.ID())
	aqe := auditor.NewQueryExecutor()
	logger.Debugf("AuditView: [%s] get query executor...done", tx.ID())
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

	logger.Debugf("AuditView: Approve... [%s]", tx.ID())
	res, err := context.RunView(ttx.NewAuditApproveView(w, tx))
	logger.Debugf("AuditView: Approve...done [%s]", tx.ID())
	return res, err
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

type CurrentHolding struct {
	EnrollmentID string `json:"enrollment_id"`
	TokenType    string `json:"token_type"`
}

// CurrentHoldingView is used to retrieve the current holding of token type of the passed enrollment id
type CurrentHoldingView struct {
	*CurrentHolding
}

func (r *CurrentHoldingView) Call(context view.Context) (interface{}, error) {
	w := ttx.MyAuditorWallet(context)
	assert.NotNil(w, "failed getting default auditor wallet")

	auditor := ttx.NewAuditor(context, w)

	aqe := auditor.NewQueryExecutor()
	defer aqe.Done()

	filter, err := aqe.NewHoldingsFilter().ByEnrollmentId(r.EnrollmentID).ByType(r.TokenType).Execute()
	assert.NoError(err, "failed retrieving holding for [%s][%s]", r.EnrollmentID, r.TokenType)
	currentHolding := filter.Sum()
	decimal := currentHolding.Decimal()
	logger.Debugf("Current Holding: [%s][%s][%s]", r.EnrollmentID, r.TokenType, decimal)

	return decimal, nil
}

type CurrentHoldingViewFactory struct{}

func (p *CurrentHoldingViewFactory) NewView(in []byte) (view.View, error) {
	f := &CurrentHoldingView{CurrentHolding: &CurrentHolding{}}
	err := json.Unmarshal(in, f.CurrentHolding)
	assert.NoError(err, "failed unmarshalling input")

	return f, nil
}

type CurrentSpending struct {
	EnrollmentID string `json:"enrollment_id"`
	TokenType    string `json:"token_type"`
}

// CurrentSpendingView is used to retrieve the current spending of token type of the passed enrollment id
type CurrentSpendingView struct {
	*CurrentSpending
}

func (r *CurrentSpendingView) Call(context view.Context) (interface{}, error) {
	w := ttx.MyAuditorWallet(context)
	assert.NotNil(w, "failed getting default auditor wallet")

	auditor := ttx.NewAuditor(context, w)

	aqe := auditor.NewQueryExecutor()
	defer aqe.Done()

	filter, err := aqe.NewPaymentsFilter().ByEnrollmentId(r.EnrollmentID).ByType(r.TokenType).Execute()
	assert.NoError(err, "failed retrieving spending for [%s][%s]", r.EnrollmentID, r.TokenType)
	currentSpending := filter.Sum()
	decimal := currentSpending.Decimal()
	logger.Debugf("Current Spending: [%s][%s][%s]", r.EnrollmentID, r.TokenType, decimal)

	return decimal, nil
}

type CurrentSpendingViewFactory struct{}

func (p *CurrentSpendingViewFactory) NewView(in []byte) (view.View, error) {
	f := &CurrentSpendingView{CurrentSpending: &CurrentSpending{}}
	err := json.Unmarshal(in, f.CurrentSpending)
	assert.NoError(err, "failed unmarshalling input")

	return f, nil
}
