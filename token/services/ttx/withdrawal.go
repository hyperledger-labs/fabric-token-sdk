/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"time"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/session"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

type WithdrawalRequest struct {
	TMSID     token.TMSID
	Recipient view.Identity
	AuditInfo []byte
	Metadata  []byte
	TokenType string
	Amount    uint64
}

type RequestWithdrawalView struct {
	Issuer    view.Identity
	TokenType string
	Amount    uint64
	TMSID     token.TMSID
	Wallet    string

	RecipientData *RecipientData
}

func NewRequestWithdrawalView(issuer view.Identity, tokenType string, amount uint64) *RequestWithdrawalView {
	return &RequestWithdrawalView{Issuer: issuer, TokenType: tokenType, Amount: amount}
}

func RequestWithdrawal(context view.Context, issuer view.Identity, wallet string, tokenType string, amount uint64, opts ...token.ServiceOption) (view.Identity, view.Session, error) {
	tmsID, err := compileServiceOptions(opts...)
	if err != nil {
		return nil, nil, errors.WithMessagef(err, "failed to compile options")
	}
	resultBoxed, err := context.RunView(NewRequestWithdrawalView(issuer, tokenType, amount).WithWallet(wallet).WithTMSID(*tmsID))
	if err != nil {
		return nil, nil, err
	}
	result := resultBoxed.([]interface{})
	ir := result[0].(*WithdrawalRequest)
	return ir.Recipient, result[1].(view.Session), nil
}

func RequestWithdrawalForRecipient(context view.Context, issuer view.Identity, wallet string, tokenType string, amount uint64, recipientData *RecipientData, opts ...token.ServiceOption) (view.Identity, view.Session, error) {
	tmsID, err := compileServiceOptions(opts...)
	if err != nil {
		return nil, nil, errors.WithMessagef(err, "failed to compile options")
	}
	resultBoxed, err := context.RunView(NewRequestWithdrawalView(issuer, tokenType, amount).WithWallet(wallet).WithTMSID(*tmsID).WithRecipientIdentity(recipientData))
	if err != nil {
		return nil, nil, err
	}
	result := resultBoxed.([]interface{})
	ir := result[0].(*WithdrawalRequest)
	return ir.Recipient, result[1].(view.Session), nil
}

func (r *RequestWithdrawalView) Call(context view.Context) (interface{}, error) {
	logger.Debugf("Respond request recipient identity using wallet [%s]", r.Wallet)

	tmsID, recipientIdentity, auditInfo, metadata, err := r.GetRecipientIdentity(context)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get recipient identity")
	}
	recipientData := &WithdrawalRequest{
		TMSID:     *tmsID,
		Recipient: recipientIdentity,
		AuditInfo: auditInfo,
		Metadata:  metadata,
		TokenType: r.TokenType,
		Amount:    r.Amount,
	}

	session, err := session.NewJSON(context, context.Initiator(), r.Issuer)
	if err != nil {
		logger.Errorf("failed to get session to [%s]: [%s]", r.Issuer, err)
		return nil, errors.Wrapf(err, "failed to get session to [%s]", r.Issuer)
	}

	// Step 3: send the public key back to the invoker
	err = session.Send(recipientData)
	if err != nil {
		logger.Errorf("failed to send recipient data: [%s]", err)
		return nil, errors.Wrapf(err, "failed to send recipient data")
	}

	return []interface{}{recipientData, session.Session()}, nil
}

func (r *RequestWithdrawalView) WithWallet(wallet string) *RequestWithdrawalView {
	r.Wallet = wallet
	return r
}

func (r *RequestWithdrawalView) WithTMSID(id token.TMSID) *RequestWithdrawalView {
	r.TMSID = id
	return r
}

func (r *RequestWithdrawalView) WithRecipientIdentity(data *RecipientData) *RequestWithdrawalView {
	r.RecipientData = data
	return r
}

func (r *RequestWithdrawalView) GetRecipientIdentity(context view.Context) (*token.TMSID, view.Identity, []byte, []byte, error) {
	if r.RecipientData != nil {
		tmsID := token.GetManagementService(context, token.WithTMSID(r.TMSID)).ID()
		return &tmsID, r.RecipientData.Identity, r.RecipientData.AuditInfo, r.RecipientData.Metadata, nil
	}

	w := GetWallet(
		context,
		r.Wallet,
		token.WithTMSID(r.TMSID),
	)
	if w == nil {
		logger.Errorf("failed to get wallet [%s]", r.Wallet)
		return nil, nil, nil, nil, errors.Errorf("wallet [%s:%s] not found", r.Wallet, r.TMSID)
	}
	recipientIdentity, err := w.GetRecipientIdentity()
	if err != nil {
		logger.Errorf("failed to get recipient identity: [%s]", err)
		return nil, nil, nil, nil, errors.Wrapf(err, "failed to get recipient identity")
	}
	auditInfo, err := w.GetAuditInfo(recipientIdentity)
	if err != nil {
		logger.Errorf("failed to get audit info: [%s]", err)
		return nil, nil, nil, nil, errors.Wrapf(err, "failed to get audit info")
	}
	metadata, err := w.GetTokenMetadata(recipientIdentity)
	if err != nil {
		logger.Errorf("failed to get token metadata: [%s]", err)
		return nil, nil, nil, nil, errors.Wrapf(err, "failed to get token metadata")
	}

	tmsID := w.TMS().ID()
	return &tmsID, recipientIdentity, auditInfo, metadata, nil
}

type ReceiveWithdrawalRequestView struct{}

func NewReceiveIssuanceRequestView() *ReceiveWithdrawalRequestView {
	return &ReceiveWithdrawalRequestView{}
}

func ReceiveWithdrawalRequest(context view.Context) (*WithdrawalRequest, error) {
	requestBoxed, err := context.RunView(NewReceiveIssuanceRequestView())
	if err != nil {
		return nil, err
	}
	ir := requestBoxed.(*WithdrawalRequest)
	return ir, nil
}

func (r *ReceiveWithdrawalRequestView) Call(context view.Context) (interface{}, error) {
	session := session.JSON(context)
	request := &WithdrawalRequest{}
	assert.NoError(session.ReceiveWithTimeout(request, 1*time.Minute), "failed to receive the withdrawal request")

	tms := token.GetManagementService(context, token.WithTMSID(request.TMSID))
	assert.NotNil(tms, "tms not found for [%s]", request.TMSID)

	if err := tms.WalletManager().RegisterRecipientIdentity(request.Recipient, request.AuditInfo, request.Metadata); err != nil {
		logger.Errorf("failed to register recipient identity: [%s]", err)
		return nil, errors.Wrapf(err, "failed to register recipient identity")
	}

	// Update the Endpoint Resolver
	caller := context.Session().Info().Caller
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("update endpoint resolver for [%s], bind to [%s]", request.Recipient, caller)
	}
	if err := view2.GetEndpointService(context).Bind(caller, request.Recipient); err != nil {
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("failed binding [%s] to [%s]", request.Recipient, caller)
		}
		return nil, errors.Wrapf(err, "failed binding [%s] to [%s]", request.Recipient, caller)
	}

	return request, nil
}
