/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"bytes"
	"time"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/session"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

type ConversionAgreement struct {
	Challenge []byte
	TMSID     token.TMSID
}

type ConversionRequest struct {
	ID            []byte
	TMSID         token.TMSID
	RecipientData RecipientData
	Tokens        []token2.LedgerToken
	Proof         []byte
	NotAnonymous  bool
}

// ConversionInitiatorView is the initiator view to request an issuer the conversion of tokens.
// The view prepares an instance of ConversionRequest and send it to the issuer.
type ConversionInitiatorView struct {
	Issuer       view.Identity
	TokenType    token2.Type
	Amount       uint64
	TMSID        token.TMSID
	Wallet       string
	NotAnonymous bool

	RecipientData *RecipientData
	Tokens        []token2.LedgerToken
}

func NewRequestConversionView(issuer view.Identity, wallet string, tokens []token2.LedgerToken, notAnonymous bool) *ConversionInitiatorView {
	return &ConversionInitiatorView{Issuer: issuer, Wallet: wallet, Tokens: tokens, NotAnonymous: notAnonymous}
}

// RequestConversion runs ConversionInitiatorView with the passed arguments.
// The view will generate a recipient identity and pass it to the issuer.
func RequestConversion(context view.Context, issuer view.Identity, wallet string, tokens []token2.LedgerToken, notAnonymous bool, opts ...token.ServiceOption) (view.Identity, view.Session, error) {
	return RequestConversionForRecipient(context, issuer, wallet, tokens, notAnonymous, nil, opts...)
}

// RequestConversionForRecipient runs ConversionInitiatorView with the passed arguments.
// The view will send the passed recipient data to the issuer.
func RequestConversionForRecipient(context view.Context, issuer view.Identity, wallet string, tokens []token2.LedgerToken, notAnonymous bool, recipientData *RecipientData, opts ...token.ServiceOption) (view.Identity, view.Session, error) {
	options, err := CompileServiceOptions(opts...)
	if err != nil {
		return nil, nil, errors.WithMessagef(err, "failed to compile options")
	}
	resultBoxed, err := context.RunView(NewRequestConversionView(
		issuer,
		wallet,
		tokens,
		notAnonymous,
	).WithWallet(wallet).WithTMSID(options.TMSID()).WithRecipientData(recipientData))
	if err != nil {
		return nil, nil, err
	}
	result := resultBoxed.([]interface{})
	ir := result[0].(*ConversionRequest)
	return ir.RecipientData.Identity, result[1].(view.Session), nil
}

func (r *ConversionInitiatorView) Call(context view.Context) (interface{}, error) {
	span := context.StartSpan("conversion_request_view")
	defer span.End()
	logger.Debugf("Respond request recipient identity using wallet [%s]", r.Wallet)

	span.AddEvent("start_session")
	session, err := session.NewJSON(context, context.Initiator(), r.Issuer)
	if err != nil {
		logger.Errorf("failed to get session to [%s]: [%s]", r.Issuer, err)
		return nil, errors.Wrapf(err, "failed to get session to [%s]", r.Issuer)
	}

	// first agreement
	agreement := &ConversionAgreement{}
	span.AddEvent("send_conversion_agreement")
	err = session.SendWithContext(context.Context(), agreement)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to send recipient data")
	}

	if err := session.ReceiveWithTimeout(agreement, 1*time.Minute); err != nil {
		return nil, errors.Wrapf(err, "failed to receive conversion agreement")
	}

	// prepare request
	span.AddEvent("send_conversion_request")

	// - recipient identity
	tmsID, recipientIdentity, auditInfo, tokenMetadata, err := r.getRecipientIdentity(context)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get recipient identity")
	}
	// - proof
	tms := token.GetManagementService(context, token.WithTMSID(*tmsID))
	if tms == nil {
		return nil, errors.Errorf("tms not found for [%s]", tmsID)
	}
	proof, err := tms.TokensService().GenConversionProof(agreement.Challenge, r.Tokens)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to generate proof")
	}

	wr := &ConversionRequest{
		ID:    agreement.Challenge,
		TMSID: *tmsID,
		RecipientData: RecipientData{
			Identity:      recipientIdentity,
			AuditInfo:     auditInfo,
			TokenMetadata: tokenMetadata,
		},
		Tokens:       r.Tokens,
		Proof:        proof,
		NotAnonymous: r.NotAnonymous,
	}
	err = session.SendWithContext(context.Context(), wr)
	if err != nil {
		logger.Errorf("failed to send recipient data: [%s]", err)
		return nil, errors.Wrapf(err, "failed to send recipient data")
	}

	return []interface{}{wr, session.Session()}, nil
}

// WithWallet sets the wallet to use to retrieve a recipient identity if it has not been passed already
func (r *ConversionInitiatorView) WithWallet(wallet string) *ConversionInitiatorView {
	r.Wallet = wallet
	return r
}

// WithTMSID sets the TMS ID to be used
func (r *ConversionInitiatorView) WithTMSID(id token.TMSID) *ConversionInitiatorView {
	r.TMSID = id
	return r
}

// WithRecipientData sets the recipient data to use
func (r *ConversionInitiatorView) WithRecipientData(data *RecipientData) *ConversionInitiatorView {
	r.RecipientData = data
	return r
}

func (r *ConversionInitiatorView) getRecipientIdentity(context view.Context) (*token.TMSID, view.Identity, []byte, []byte, error) {
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

// ConversionResponderView this is the view used by the issuer to receive a conversion request
type ConversionResponderView struct{}

func NewReceiveConversionRequestView() *ConversionResponderView {
	return &ConversionResponderView{}
}

func ReceiveConversionRequest(context view.Context) (*ConversionRequest, error) {
	requestBoxed, err := context.RunView(NewReceiveConversionRequestView())
	if err != nil {
		return nil, err
	}
	ir := requestBoxed.(*ConversionRequest)
	return ir, nil
}

func (r *ConversionResponderView) Call(context view.Context) (interface{}, error) {
	span := context.StartSpan("receive_conversion_request_view")
	defer span.End()

	session := session.JSON(context)
	agreement := &ConversionAgreement{}
	if err := session.ReceiveWithTimeout(agreement, 1*time.Minute); err != nil {
		return nil, errors.Wrapf(err, "failed to receive conversion request")
	}
	span.AddEvent("received_conversion_request")
	// sample agreement ID
	tms := token.GetManagementService(context, token.WithTMSID(agreement.TMSID))
	if tms == nil {
		return nil, errors.Errorf("tms not found for [%s]", agreement.TMSID)
	}
	conversionChallange, err := tms.TokensService().NewConversionChallenge()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to generate conversion challenge")
	}
	agreement.Challenge = conversionChallange
	agreement.TMSID = tms.ID()

	// send the agreement back
	if err := session.Send(agreement); err != nil {
		return nil, errors.Wrapf(err, "failed to send conversion request")
	}
	// receive the response
	request := &ConversionRequest{}
	if err := session.ReceiveWithTimeout(request, 1*time.Minute); err != nil {
		return nil, errors.Wrapf(err, "failed to receive conversion request")
	}

	// check the ID is the same
	if !bytes.Equal(agreement.Challenge, request.ID) {
		return nil, errors.Errorf("agreement ID mismatch [%v] != [%v]", agreement.Challenge, request.ID)
	}
	// check the TMS is the same
	if !agreement.TMSID.Equal(request.TMSID) {
		return nil, errors.Errorf("agreement TMSID mismatch [%v] != [%v]", agreement.TMSID, request.TMSID)
	}

	if err := tms.WalletManager().RegisterRecipientIdentity(&request.RecipientData); err != nil {
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
