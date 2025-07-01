/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"bytes"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/endpoint"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/json/session"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type UpgradeTokensAgreement struct {
	Challenge []byte
	TMSID     token.TMSID
}

type UpgradeTokensRequest struct {
	TMSID token.TMSID // The TMSID this request refers to

	ID     token.TokensUpgradeChallenge // The unique ID of this request, it used as challenge of the upgrade protocol
	Tokens []token2.LedgerToken         // The tokens to be upgraded
	Proof  token.TokensUpgradeProof     // The proof

	RecipientData RecipientData // The info about the recipient of the new issues tokens
	NotAnonymous  bool          // Should be the transaction anonymous
}

// UpgradeTokensInitiatorView is the initiator view to request an issuer the upgrade of tokens.
// The view prepares an instance of UpgradeTokensRequest and send it to the issuer.
type UpgradeTokensInitiatorView struct {
	Issuer       view.Identity
	TokenType    token2.Type
	Amount       uint64
	TMSID        token.TMSID
	Wallet       string
	NotAnonymous bool

	RecipientData *RecipientData
	Tokens        []token2.LedgerToken
}

func NewRequestTokensUpgradeView(issuer view.Identity, wallet string, tokens []token2.LedgerToken, notAnonymous bool) *UpgradeTokensInitiatorView {
	return &UpgradeTokensInitiatorView{Issuer: issuer, Wallet: wallet, Tokens: tokens, NotAnonymous: notAnonymous}
}

// RequestTokensUpgrade runs UpgradeTokensInitiatorView with the passed arguments.
// The view will generate a recipient identity and pass it to the issuer.
func RequestTokensUpgrade(context view.Context, issuer view.Identity, wallet string, tokens []token2.LedgerToken, notAnonymous bool, opts ...token.ServiceOption) (view.Identity, view.Session, error) {
	return RequestTokensUpgradeForRecipient(context, issuer, wallet, tokens, notAnonymous, nil, opts...)
}

// RequestTokensUpgradeForRecipient runs UpgradeTokensInitiatorView with the passed arguments.
// The view will send the passed recipient data to the issuer.
func RequestTokensUpgradeForRecipient(context view.Context, issuer view.Identity, wallet string, tokens []token2.LedgerToken, notAnonymous bool, recipientData *RecipientData, opts ...token.ServiceOption) (view.Identity, view.Session, error) {
	options, err := CompileServiceOptions(opts...)
	if err != nil {
		return nil, nil, errors.WithMessagef(err, "failed to compile options")
	}
	resultBoxed, err := context.RunView(NewRequestTokensUpgradeView(
		issuer,
		wallet,
		tokens,
		notAnonymous,
	).WithWallet(wallet).WithTMSID(options.TMSID()).WithRecipientData(recipientData))
	if err != nil {
		return nil, nil, err
	}
	result := resultBoxed.([]interface{})
	ir := result[0].(*UpgradeTokensRequest)
	return ir.RecipientData.Identity, result[1].(view.Session), nil
}

func (r *UpgradeTokensInitiatorView) Call(context view.Context) (interface{}, error) {

	logger.DebugfContext(context.Context(), "Respond request recipient identity using wallet [%s]", r.Wallet)

	session, err := session.NewJSON(context, context.Initiator(), r.Issuer)
	if err != nil {
		logger.Errorf("failed to get session to [%s]: [%s]", r.Issuer, err)
		return nil, errors.Wrapf(err, "failed to get session to [%s]", r.Issuer)
	}

	// first agreement
	agreement := &UpgradeTokensAgreement{}
	logger.DebugfContext(context.Context(), "Send upgrade agreement")
	err = session.SendWithContext(context.Context(), agreement)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to send recipient data")
	}

	if err := session.ReceiveWithTimeout(agreement, 1*time.Minute); err != nil {
		return nil, errors.Wrapf(err, "failed to receive upgrade agreement")
	}

	// prepare request
	logger.DebugfContext(context.Context(), "Send upgrade request")

	// - recipient identity
	tmsID, recipientData, err := r.getRecipientData(context)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get recipient identity")
	}
	// - proof
	tms := token.GetManagementService(context, token.WithTMSID(*tmsID))
	if tms == nil {
		return nil, errors.Errorf("tms not found for [%s]", tmsID)
	}
	proof, err := tms.TokensService().GenUpgradeProof(agreement.Challenge, r.Tokens)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to generate proof")
	}

	wr := &UpgradeTokensRequest{
		ID:            agreement.Challenge,
		TMSID:         *tmsID,
		RecipientData: *recipientData,
		Tokens:        r.Tokens,
		Proof:         proof,
		NotAnonymous:  r.NotAnonymous,
	}
	err = session.SendWithContext(context.Context(), wr)
	if err != nil {
		logger.Errorf("failed to send recipient data: [%s]", err)
		return nil, errors.Wrapf(err, "failed to send recipient data")
	}

	return []interface{}{wr, session.Session()}, nil
}

// WithWallet sets the wallet to use to retrieve a recipient identity if it has not been passed already
func (r *UpgradeTokensInitiatorView) WithWallet(wallet string) *UpgradeTokensInitiatorView {
	r.Wallet = wallet
	return r
}

// WithTMSID sets the TMS ID to be used
func (r *UpgradeTokensInitiatorView) WithTMSID(id token.TMSID) *UpgradeTokensInitiatorView {
	r.TMSID = id
	return r
}

// WithRecipientData sets the recipient data to use
func (r *UpgradeTokensInitiatorView) WithRecipientData(data *RecipientData) *UpgradeTokensInitiatorView {
	r.RecipientData = data
	return r
}

func (r *UpgradeTokensInitiatorView) getRecipientData(context view.Context) (*token.TMSID, *RecipientData, error) {
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

// UpgradeTokensResponderView this is the view used by the issuer to receive a upgrade request
type UpgradeTokensResponderView struct{}

func NewReceiveUpgradeRequestView() *UpgradeTokensResponderView {
	return &UpgradeTokensResponderView{}
}

func ReceiveTokensUpgradeRequest(context view.Context) (*UpgradeTokensRequest, error) {
	requestBoxed, err := context.RunView(NewReceiveUpgradeRequestView())
	if err != nil {
		return nil, err
	}
	ir := requestBoxed.(*UpgradeTokensRequest)
	return ir, nil
}

func (r *UpgradeTokensResponderView) Call(context view.Context) (interface{}, error) {
	session := session.JSON(context)
	agreement := &UpgradeTokensAgreement{}
	if err := session.ReceiveWithTimeout(agreement, 1*time.Minute); err != nil {
		return nil, errors.Wrapf(err, "failed to receive upgrade request")
	}
	logger.DebugfContext(context.Context(), "Received upgrade request")
	// sample agreement ID
	tms := token.GetManagementService(context, token.WithTMSID(agreement.TMSID))
	if tms == nil {
		return nil, errors.Errorf("tms not found for [%s]", agreement.TMSID)
	}
	challenge, err := tms.TokensService().NewUpgradeChallenge()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to generate upgrade challenge")
	}
	agreement.Challenge = challenge
	agreement.TMSID = tms.ID()

	// send the agreement back
	if err := session.Send(agreement); err != nil {
		return nil, errors.Wrapf(err, "failed to send upgrade request")
	}
	// receive the response
	request := &UpgradeTokensRequest{}
	if err := session.ReceiveWithTimeout(request, 1*time.Minute); err != nil {
		return nil, errors.Wrapf(err, "failed to receive upgrade request")
	}

	// check the ID is the same
	if !bytes.Equal(agreement.Challenge, request.ID) {
		return nil, errors.Errorf("agreement ID mismatch [%v] != [%v]", agreement.Challenge, request.ID)
	}
	// check the TMS is the same
	if !agreement.TMSID.Equal(request.TMSID) {
		return nil, errors.Errorf("agreement TMSID mismatch [%v] != [%v]", agreement.TMSID, request.TMSID)
	}

	// register recipient data
	if err := tms.WalletManager().RegisterRecipientIdentity(&request.RecipientData); err != nil {
		return nil, errors.Wrapf(err, "failed to register recipient identity")
	}

	// Update the Endpoint Resolver
	caller := context.Session().Info().Caller
	logger.Debugf("update endpoint resolver for [%s], bind to [%s]", request.RecipientData.Identity, caller)
	if err := endpoint.GetService(context).Bind(context.Context(), caller, request.RecipientData.Identity); err != nil {
		logger.Debugf("failed binding [%s] to [%s]", request.RecipientData.Identity, caller)
		return nil, errors.Wrapf(err, "failed binding [%s] to [%s]", request.RecipientData.Identity, caller)
	}

	return request, nil
}
