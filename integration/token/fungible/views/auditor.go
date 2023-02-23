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
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
)

type AuditView struct{}

func (a *AuditView) Call(context view.Context) (interface{}, error) {
	logger.Debugf("AuditView: [%s]", context.ID())
	tx, err := ttx.ReceiveTransaction(context, ttx.WithNoTransactionVerification())

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
	defer auditor.Release(tx)

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
			if tokenType == "MAX" {
				continue
			}
			// compute the payment done in the transaction
			sent := inputs.ByEnrollmentID(eID).ByType(tokenType).Sum()
			received := outputs.ByEnrollmentID(eID).ByType(tokenType).Sum()
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
			if tokenType == "MAX" {
				continue
			}
			// compute the payment done in the transaction
			sent := inputs.ByEnrollmentID(eID).ByType(tokenType).Sum()
			received := outputs.ByEnrollmentID(eID).ByType(tokenType).Sum()
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
			fmt.Printf("Cumulative Limit: [%s] Last NewPaymentsFilter [%s], type [%s]\n", eID, sumLastPayments.Text(10), tokenType)

			// R3: The default configuration is customized by a specific organisation (Guarantor)
			total := sumLastPayments.Add(sumLastPayments, diff)
			assert.True(total.Cmp(big.NewInt(2000)) < 0, "cumulative payment limit reached [%s][%s][%s]", eID, tokenType, total.Text(10))
		}
	}

	// R4: Default holding limit is set to 3000.
	eIDs = outputs.EnrollmentIDs()
	tokenTypes = outputs.TokenTypes()
	for _, eID := range eIDs {
		assert.NotEmpty(eID, "enrollment id should not be empty")
		for _, tokenType := range tokenTypes {
			if tokenType == "MAX" {
				continue
			}
			// compute the amount received
			received := outputs.ByEnrollmentID(eID).ByType(tokenType).Sum()
			sent := inputs.ByEnrollmentID(eID).ByType(tokenType).Sum()
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

			fmt.Printf("Holding Limit: [%s] Current [%s], type [%s]\n", eID, currentHolding.Text(10), tokenType)

			total := currentHolding.Add(currentHolding, diff)
			assert.True(total.Cmp(big.NewInt(3000)) < 0, "holding limit reached [%s][%s][%s]", eID, tokenType, total.Text(10))
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
	EnrollmentID string
	TokenType    string
	TMSID        token.TMSID
}

// CurrentHoldingView is used to retrieve the current holding of token type of the passed enrollment id
type CurrentHoldingView struct {
	*CurrentHolding
}

func (r *CurrentHoldingView) Call(context view.Context) (interface{}, error) {
	tms := token.GetManagementService(context, token.WithTMSID(r.TMSID))
	assert.NotNil(tms, "tms not found [%s]", r.TMSID)

	w := tms.WalletManager().AuditorWallet("")
	assert.NotNil(w, "failed getting default auditor wallet")

	auditor := ttx.NewAuditor(context, w)

	aqe := auditor.NewQueryExecutor()
	defer aqe.Done()

	filter, err := aqe.NewHoldingsFilter().ByEnrollmentId(r.EnrollmentID).ByType(r.TokenType).Execute()
	assert.NoError(err, "failed retrieving holding for [%s][%s]", r.EnrollmentID, r.TokenType)
	currentHolding := filter.Sum()
	decimal := currentHolding.Text(10)
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
	decimal := currentSpending.Text(10)
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

type SetTransactionAuditStatus struct {
	TxID   string
	Status ttx.TxStatus
}

// SetTransactionAuditStatusView is used to set the status of a given transaction in the audit db
type SetTransactionAuditStatusView struct {
	*SetTransactionAuditStatus
}

func (r *SetTransactionAuditStatusView) Call(context view.Context) (interface{}, error) {
	w := ttx.MyAuditorWallet(context)
	assert.NotNil(w, "failed getting default auditor wallet")

	auditor := ttx.NewAuditor(context, w)
	assert.NoError(auditor.SetStatus(r.TxID, r.Status), "failed to set status of [%s] to [%d]", r.TxID, r.Status)

	if r.Status == ttxdb.Deleted {
		tms := token.GetManagementService(context)
		assert.NotNil(tms, "failed to get default tms")
		net := network.GetInstance(context, tms.Network(), tms.Channel())
		assert.NotNil(net, "failed to get network [%s:%s]", tms.Network(), tms.Channel())
		v, err := net.Vault(tms.Namespace())
		assert.NoError(err, "failed to get vault [%s:%s:%s]", tms.Network(), tms.Channel(), tms.Namespace())
		assert.NoError(v.DiscardTx(r.TxID), "failed to discard tx [%s:%s:%s:%s]", tms.Network(), tms.Channel(), tms.Namespace(), r.TxID)
	}

	return nil, nil
}

type SetTransactionAuditStatusViewFactory struct{}

func (p *SetTransactionAuditStatusViewFactory) NewView(in []byte) (view.View, error) {
	f := &SetTransactionAuditStatusView{SetTransactionAuditStatus: &SetTransactionAuditStatus{}}
	err := json.Unmarshal(in, f.SetTransactionAuditStatus)
	assert.NoError(err, "failed unmarshalling input")

	return f, nil
}

type GetAuditorWalletIdentity struct {
	ID string
}

type GetAuditorWalletIdentityView struct {
	*GetAuditorWalletIdentity
}

type GetAuditorWalletIdentityViewFactory struct{}

func (g *GetAuditorWalletIdentity) Call(context view.Context) (interface{}, error) {
	defaultAuditorWallet := token.GetManagementService(context).WalletManager().AuditorWallet(g.ID)
	assert.NotNil(defaultAuditorWallet, "no default auditor wallet")
	id, err := defaultAuditorWallet.GetAuditorIdentity()
	assert.NoError(err, "failed getting auditorIdentity ")
	return id, err
}

func (p *GetAuditorWalletIdentityViewFactory) NewView(in []byte) (view.View, error) {
	f := &GetAuditorWalletIdentityView{GetAuditorWalletIdentity: &GetAuditorWalletIdentity{}}
	err := json.Unmarshal(in, f.GetAuditorWalletIdentity)
	assert.NoError(err, "failed unmarshalling input")
	return f, nil
}
