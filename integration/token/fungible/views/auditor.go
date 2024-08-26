/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package views

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
)

type AuditView struct {
	*token.TMSID
}

func (a *AuditView) Call(context view.Context) (interface{}, error) {
	logger.Debugf("AuditView: [%s]", context.ID())
	tx, err := ttx.ReceiveTransaction(context, append(TxOpts(a.TMSID), ttx.WithNoTransactionVerification())...)
	assert.NoError(err, "failed receiving transaction")
	logger.Debugf("AuditView: [%s]", tx.ID())

	w := ttx.MyAuditorWallet(context, ServiceOpts(a.TMSID)...)
	assert.NotNil(w, "failed getting default auditor wallet")

	logger.Debugf("AuditView: Approve... [%s]", tx.ID())
	res, err := context.RunView(ttx.NewAuditApproveView(w, tx))
	logger.Debugf("AuditView: Approve...done [%s]", tx.ID())
	return res, err
}

type RegisterAuditor struct {
	*token.TMSID
}

type RegisterAuditorView struct {
	*RegisterAuditor
}

func (r *RegisterAuditorView) Call(context view.Context) (interface{}, error) {
	return context.RunView(ttx.NewRegisterAuditorView(
		&AuditView{r.TMSID},
		ServiceOpts(r.TMSID)...,
	))
}

type RegisterAuditorViewFactory struct{}

func (p *RegisterAuditorViewFactory) NewView(in []byte) (view.View, error) {
	f := &RegisterAuditorView{RegisterAuditor: &RegisterAuditor{}}
	if in != nil {
		err := json.Unmarshal(in, f.RegisterAuditor)
		assert.NoError(err, "failed unmarshalling input")
	}
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

	auditor, err := ttx.NewAuditor(context, w)
	assert.NoError(err, "failed to get auditor instance")

	filter, err := auditor.NewHoldingsFilter().ByEnrollmentId(r.EnrollmentID).ByType(r.TokenType).Execute()
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
	EnrollmentID string       `json:"enrollment_id"`
	TokenType    string       `json:"token_type"`
	TMSID        *token.TMSID `json:"tmsid"`
}

// CurrentSpendingView is used to retrieve the current spending of token type of the passed enrollment id
type CurrentSpendingView struct {
	*CurrentSpending
}

func (r *CurrentSpendingView) Call(context view.Context) (interface{}, error) {
	w := ttx.MyAuditorWallet(context, ServiceOpts(r.TMSID)...)
	assert.NotNil(w, "failed getting default auditor wallet")

	auditor, err := ttx.NewAuditor(context, w)
	assert.NoError(err, "failed to get auditor instance")

	filter, err := auditor.NewPaymentsFilter().ByEnrollmentId(r.EnrollmentID).ByType(r.TokenType).Execute()
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
	TxID    string
	Status  ttx.TxStatus
	Message string
}

// SetTransactionAuditStatusView is used to set the status of a given transaction in the audit db
type SetTransactionAuditStatusView struct {
	*SetTransactionAuditStatus
}

func (r *SetTransactionAuditStatusView) Call(context view.Context) (interface{}, error) {
	w := ttx.MyAuditorWallet(context)
	assert.NotNil(w, "failed getting default auditor wallet")

	auditor, err := ttx.NewAuditor(context, w)
	assert.NoError(err, "failed to get auditor instance")
	assert.NoError(auditor.SetStatus(context.Context(), r.TxID, r.Status, r.Message), "failed to set status of [%s] to [%d]", r.TxID, r.Status)

	if r.Status == ttx.Deleted {
		tms := token.GetManagementService(context)
		assert.NotNil(tms, "failed to get default tms")
		net := network.GetInstance(context, tms.Network(), tms.Channel())
		assert.NotNil(net, "failed to get network [%s:%s]", tms.Network(), tms.Channel())
		v, err := net.Vault(tms.Namespace())
		assert.NoError(err)
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

func (g *GetAuditorWalletIdentity) Call(context view.Context) (interface{}, error) {
	defaultAuditorWallet := token.GetManagementService(context).WalletManager().AuditorWallet(g.ID)
	assert.NotNil(defaultAuditorWallet, "no default auditor wallet")
	id, err := defaultAuditorWallet.GetAuditorIdentity()
	assert.NoError(err, "failed getting auditorIdentity ")
	return id, err
}

type GetAuditorWalletIdentityViewFactory struct{}

func (p *GetAuditorWalletIdentityViewFactory) NewView(in []byte) (view.View, error) {
	f := &GetAuditorWalletIdentityView{GetAuditorWalletIdentity: &GetAuditorWalletIdentity{}}
	err := json.Unmarshal(in, f.GetAuditorWalletIdentity)
	assert.NoError(err, "failed unmarshalling input")
	return f, nil
}
