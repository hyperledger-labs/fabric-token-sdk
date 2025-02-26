/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"time"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/json/session"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap/zapcore"
)

type WithdrawalRequest struct {
	TMSID         token.TMSID
	RecipientData RecipientData
	TokenType     token2.Type
	Amount        uint64
	NotAnonymous  bool
}

// RequestWithdrawalView is the initiator view to request an issuer the issuance of tokens.
// The view prepares an instance of WithdrawalRequest and send it to the issuer.
type RequestWithdrawalView struct {
	Issuer       view.Identity
	TokenType    token2.Type
	Amount       uint64
	TMSID        token.TMSID
	Wallet       string
	NotAnonymous bool

	RecipientData *RecipientData
}

func NewRequestWithdrawalView(issuer view.Identity, tokenType token2.Type, amount uint64, notAnonymous bool) *RequestWithdrawalView {
	return &RequestWithdrawalView{Issuer: issuer, TokenType: tokenType, Amount: amount, NotAnonymous: notAnonymous}
}

// RequestWithdrawal runs RequestWithdrawalView with the passed arguments.
// The view will generate a recipient identity and pass it to the issuer.
func RequestWithdrawal(context view.Context, issuer view.Identity, wallet string, tokenType token2.Type, amount uint64, notAnonymous bool, opts ...token.ServiceOption) (view.Identity, view.Session, error) {
	return RequestWithdrawalForRecipient(context, issuer, wallet, tokenType, amount, notAnonymous, nil, opts...)
}

// RequestWithdrawalForRecipient runs RequestWithdrawalView with the passed arguments.
// The view will send the passed recipient data to the issuer.
func RequestWithdrawalForRecipient(context view.Context, issuer view.Identity, wallet string, tokenType token2.Type, amount uint64, notAnonymous bool, recipientData *RecipientData, opts ...token.ServiceOption) (view.Identity, view.Session, error) {
	options, err := CompileServiceOptions(opts...)
	if err != nil {
		return nil, nil, errors.WithMessagef(err, "failed to compile options")
	}
	resultBoxed, err := context.RunView(NewRequestWithdrawalView(issuer, tokenType, amount, notAnonymous).WithWallet(wallet).WithTMSID(options.TMSID()).WithRecipientData(recipientData))
	if err != nil {
		return nil, nil, err
	}
	result := resultBoxed.([]interface{})
	ir := result[0].(*WithdrawalRequest)
	return ir.RecipientData.Identity, result[1].(view.Session), nil
}

func (r *RequestWithdrawalView) Call(context view.Context) (interface{}, error) {
	span := trace.SpanFromContext(context.Context())

	logger.Debugf("Respond request recipient identity using wallet [%s]", r.Wallet)

	tmsID, recipientData, err := r.getRecipientIdentity(context)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get recipient data")
	}
	wr := &WithdrawalRequest{
		TMSID:         *tmsID,
		RecipientData: *recipientData,
		TokenType:     r.TokenType,
		Amount:        r.Amount,
		NotAnonymous:  r.NotAnonymous,
	}

	span.AddEvent("start_session")
	session, err := session.NewJSON(context, context.Initiator(), r.Issuer)
	if err != nil {
		logger.Errorf("failed to get session to [%s]: [%s]", r.Issuer, err)
		return nil, errors.Wrapf(err, "failed to get session to [%s]", r.Issuer)
	}

	span.AddEvent("send_withdrawal_request")
	err = session.SendWithContext(context.Context(), wr)
	if err != nil {
		logger.Errorf("failed to send recipient data: [%s]", err)
		return nil, errors.Wrapf(err, "failed to send recipient data")
	}

	return []interface{}{wr, session.Session()}, nil
}

// WithWallet sets the wallet to use to retrieve a recipient identity if it has not been passed already
func (r *RequestWithdrawalView) WithWallet(wallet string) *RequestWithdrawalView {
	r.Wallet = wallet
	return r
}

// WithTMSID sets the TMS ID to be used
func (r *RequestWithdrawalView) WithTMSID(id token.TMSID) *RequestWithdrawalView {
	r.TMSID = id
	return r
}

// WithRecipientData sets the recipient data to use
func (r *RequestWithdrawalView) WithRecipientData(data *RecipientData) *RequestWithdrawalView {
	r.RecipientData = data
	return r
}

func (r *RequestWithdrawalView) getRecipientIdentity(context view.Context) (*token.TMSID, *RecipientData, error) {
	if r.RecipientData != nil {
		tmsID := token.GetManagementService(context, token.WithTMSID(r.TMSID)).ID()
		return &tmsID, r.RecipientData, nil
	}

	w := GetWallet(
		context,
		r.Wallet,
		token.WithTMSID(r.TMSID),
	)
	if w == nil {
		logger.Errorf("failed to get wallet [%s]", r.Wallet)
		return nil, nil, errors.Errorf("wallet [%s:%s] not found", r.Wallet, r.TMSID)
	}
	recipientData, err := w.GetRecipientData()
	if err != nil {
		logger.Errorf("failed to get recipient data: [%s]", err)
		return nil, nil, errors.Wrapf(err, "failed to get recipient data")
	}

	tmsID := w.TMS().ID()
	return &tmsID, recipientData, nil
}

// ReceiveWithdrawalRequestView this is the view used by the issuer to receive a withdrawal request
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
	span := trace.SpanFromContext(context.Context())

	session := session.JSON(context)
	request := &WithdrawalRequest{}
	assert.NoError(session.ReceiveWithTimeout(request, 1*time.Minute), "failed to receive the withdrawal request")

	span.AddEvent("received_withdrawal_request")
	tms := token.GetManagementService(context, token.WithTMSID(request.TMSID))
	assert.NotNil(tms, "tms not found for [%s]", request.TMSID)

	if err := tms.WalletManager().RegisterRecipientIdentity(&request.RecipientData); err != nil {
		logger.Errorf("failed to register recipient identity: [%s]", err)
		return nil, errors.Wrapf(err, "failed to register recipient identity")
	}

	// Update the Endpoint Resolver
	caller := context.Session().Info().Caller
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("update endpoint resolver for [%s], bind to [%s]", request.RecipientData.Identity, caller)
	}
	if err := view2.GetEndpointService(context).Bind(caller, request.RecipientData.Identity); err != nil {
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("failed binding [%s] to [%s]", request.RecipientData.Identity, caller)
		}
		return nil, errors.Wrapf(err, "failed binding [%s] to [%s]", request.RecipientData.Identity, caller)
	}

	return request, nil
}
