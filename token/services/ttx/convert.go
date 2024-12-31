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
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

type ConversionRequest struct {
	TMSID             token.TMSID
	RecipientData     RecipientData
	UnspendableTokens []token2.UnspendableTokenInWallet
	NotAnonymous      bool
}

// RequestConversionView is the initiator view to request an issuer the issuance of tokens.
// The view prepares an instance of ConversionRequest and send it to the issuer.
type RequestConversionView struct {
	Issuer       view.Identity
	TokenType    string
	Amount       uint64
	TMSID        token.TMSID
	Wallet       string
	NotAnonymous bool

	RecipientData            *RecipientData
	UnspendableTokenInWallet []token2.UnspendableTokenInWallet
}

func NewRequestConversionView(issuer view.Identity, wallet string, unspendableTokenInWallet []token2.UnspendableTokenInWallet, notAnonymous bool) *RequestConversionView {
	return &RequestConversionView{Issuer: issuer, Wallet: wallet, UnspendableTokenInWallet: unspendableTokenInWallet, NotAnonymous: notAnonymous}
}

// RequestConversion runs RequestConversionView with the passed arguments.
// The view will generate a recipient identity and pass it to the issuer.
func RequestConversion(context view.Context, issuer view.Identity, wallet string, unspendableTokenInWallet []token2.UnspendableTokenInWallet, notAnonymous bool, opts ...token.ServiceOption) (view.Identity, view.Session, error) {
	return RequestConversionForRecipient(context, issuer, wallet, unspendableTokenInWallet, notAnonymous, nil, opts...)
}

// RequestConversionForRecipient runs RequestConversionView with the passed arguments.
// The view will send the passed recipient data to the issuer.
func RequestConversionForRecipient(context view.Context, issuer view.Identity, wallet string, unspendableTokenInWallet []token2.UnspendableTokenInWallet, notAnonymous bool, recipientData *RecipientData, opts ...token.ServiceOption) (view.Identity, view.Session, error) {
	options, err := CompileServiceOptions(opts...)
	if err != nil {
		return nil, nil, errors.WithMessagef(err, "failed to compile options")
	}
	resultBoxed, err := context.RunView(NewRequestConversionView(
		issuer,
		wallet,
		unspendableTokenInWallet,
		notAnonymous,
	).WithWallet(wallet).WithTMSID(options.TMSID()).WithRecipientData(recipientData))
	if err != nil {
		return nil, nil, err
	}
	result := resultBoxed.([]interface{})
	ir := result[0].(*ConversionRequest)
	return ir.RecipientData.Identity, result[1].(view.Session), nil
}

func (r *RequestConversionView) Call(context view.Context) (interface{}, error) {
	span := context.StartSpan("conversion_request_view")
	defer span.End()
	logger.Debugf("Respond request recipient identity using wallet [%s]", r.Wallet)

	tmsID, recipientIdentity, auditInfo, tokenMetadata, err := r.getRecipientIdentity(context)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get recipient identity")
	}
	wr := &ConversionRequest{
		TMSID: *tmsID,
		RecipientData: RecipientData{
			Identity:      recipientIdentity,
			AuditInfo:     auditInfo,
			TokenMetadata: tokenMetadata,
		},
		UnspendableTokens: r.UnspendableTokenInWallet,
		NotAnonymous:      r.NotAnonymous,
	}

	span.AddEvent("start_session")
	session, err := session.NewJSON(context, context.Initiator(), r.Issuer)
	if err != nil {
		logger.Errorf("failed to get session to [%s]: [%s]", r.Issuer, err)
		return nil, errors.Wrapf(err, "failed to get session to [%s]", r.Issuer)
	}

	span.AddEvent("send_conversion_request")
	err = session.SendWithContext(context.Context(), wr)
	if err != nil {
		logger.Errorf("failed to send recipient data: [%s]", err)
		return nil, errors.Wrapf(err, "failed to send recipient data")
	}

	return []interface{}{wr, session.Session()}, nil
}

// WithWallet sets the wallet to use to retrieve a recipient identity if it has not been passed already
func (r *RequestConversionView) WithWallet(wallet string) *RequestConversionView {
	r.Wallet = wallet
	return r
}

// WithTMSID sets the TMS ID to be used
func (r *RequestConversionView) WithTMSID(id token.TMSID) *RequestConversionView {
	r.TMSID = id
	return r
}

// WithRecipientData sets the recipient data to use
func (r *RequestConversionView) WithRecipientData(data *RecipientData) *RequestConversionView {
	r.RecipientData = data
	return r
}

func (r *RequestConversionView) getRecipientIdentity(context view.Context) (*token.TMSID, view.Identity, []byte, []byte, error) {
	if r.RecipientData != nil {
		tmsID := token.GetManagementService(context, token.WithTMSID(r.TMSID)).ID()
		return &tmsID, r.RecipientData.Identity, r.RecipientData.AuditInfo, r.RecipientData.TokenMetadata, nil
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

// ReceiveConversionRequestView this is the view used by the issuer to receive a conversion request
type ReceiveConversionRequestView struct{}

func NewReceiveConversionRequestView() *ReceiveConversionRequestView {
	return &ReceiveConversionRequestView{}
}

func ReceiveConversionRequest(context view.Context) (*ConversionRequest, error) {
	requestBoxed, err := context.RunView(NewReceiveConversionRequestView())
	if err != nil {
		return nil, err
	}
	ir := requestBoxed.(*ConversionRequest)
	return ir, nil
}

func (r *ReceiveConversionRequestView) Call(context view.Context) (interface{}, error) {
	span := context.StartSpan("receive_conversion_request_view")
	defer span.End()
	session := session.JSON(context)
	request := &ConversionRequest{}
	assert.NoError(session.ReceiveWithTimeout(request, 1*time.Minute), "failed to receive the conversion request")

	span.AddEvent("received_conversion_request")
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
