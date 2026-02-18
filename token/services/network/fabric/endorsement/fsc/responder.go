/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fsc

import (
	"context"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/services/endorser"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	tdriver "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/translator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

const (
	TransientTMSIDKey        = "tmsID"
	TransientTokenRequestKey = "token_request"

	ChaincodeVersion = "1.0"
	InvokeFunction   = "invoke"
)

type Request struct {
	Tx               *endorser.Transaction
	Rws              *fabric.RWSet
	TMSID            token2.TMSID
	Anchor           string
	RequestRaw       []byte
	Actions          []any
	Meta             map[string][]byte
	Tms              *token2.ManagementService
	PublicParamsHash tdriver.PPHash
}

type RequestApprovalResponderView struct {
	endorserService               EndorserService
	keyTranslator                 translator.KeyTranslator
	getTranslator                 TranslatorProviderFunc
	tokenManagementSystemProvider TokenManagementSystemProvider
	storageProvider               StorageProvider
}

func NewRequestApprovalResponderView(
	keyTranslator translator.KeyTranslator,
	getTranslator TranslatorProviderFunc,
	endorserService EndorserService,
	tokenManagementSystemProvider TokenManagementSystemProvider,
	storageProvider StorageProvider,
) *RequestApprovalResponderView {
	return &RequestApprovalResponderView{
		keyTranslator:                 keyTranslator,
		getTranslator:                 getTranslator,
		endorserService:               endorserService,
		tokenManagementSystemProvider: tokenManagementSystemProvider,
		storageProvider:               storageProvider,
	}
}

func (r *RequestApprovalResponderView) Call(context view.Context) (any, error) {
	// receive
	request, err := r.receive(context)
	if err != nil {
		return nil, errors.Join(ErrReceivedProposal, err)
	}
	defer request.Rws.Done()

	// validate proposal
	err = r.validateProposal(context, request)
	if err != nil {
		return nil, errors.Join(ErrValidateProposal, err)
	}

	// validate
	err = r.validate(context, request, func(id token.ID) ([]byte, error) {
		key, err := r.keyTranslator.CreateOutputKey(id.TxId, id.Index)
		if err != nil {
			return nil, errors.WithMessagef(err, "failed to create token key for id [%s]", id)
		}

		return request.Rws.GetDirectState(request.TMSID.Namespace, key)
	})
	if err != nil {
		return nil, errors.Join(ErrValidateProposal, err)
	}

	// endorse
	res, err := r.endorse(context, request)
	if err != nil {
		return nil, errors.Join(ErrEndorseProposal, err)
	}

	return res, nil
}

func (r *RequestApprovalResponderView) receive(ctx view.Context) (*Request, error) {
	logger.DebugfContext(ctx.Context(), "Waiting for transaction on context [%s]", ctx.ID())
	tx, err := r.endorserService.ReceiveTx(ctx)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to received transaction for approval")
	}
	logger.DebugfContext(ctx.Context(), "Received transaction [%s] for endorsement on context [%s]", tx.ID(), ctx.ID())
	defer logger.DebugfContext(ctx.Context(), "Return endorsement result for TX [%s]", tx.ID())

	// validate transient

	// check the number of transient keys
	var tmsID token2.TMSID
	if len(tx.Transaction.Transient()) != 2 {
		return nil, errors.Wrapf(ErrInvalidTransient, "invalid number of transient field, expected 2, got %d", len(tx.Transaction.Transient()))
	}

	// TMS ID
	if err := tx.GetTransientState(TransientTMSIDKey, &tmsID); err != nil {
		return nil, errors.Wrapf(errors.Join(err, ErrInvalidTransient), "empty tms id")
	}
	if len(tmsID.Network) == 0 || len(tmsID.Channel) == 0 || len(tmsID.Namespace) == 0 {
		return nil, errors.Wrapf(errors.Join(err, ErrInvalidTransient), "invalid tms id [%s]", tmsID)
	}
	tms, err := r.tokenManagementSystemProvider.GetManagementService(token2.WithTMSID(tmsID))
	if err != nil {
		return nil, errors.Wrapf(errors.Join(err, ErrInvalidTransient), "cannot find TMS for [%s]", tmsID)
	}
	if !tms.ID().Equal(tmsID) {
		return nil, errors.Wrapf(errors.Join(err, ErrInvalidTransient), "tms ids do not match")
	}
	logger.DebugfContext(ctx.Context(), "evaluate token request on TMS [%s]", tmsID)

	// token request
	requestRaw := tx.GetTransient(TransientTokenRequestKey)
	if len(requestRaw) == 0 {
		return nil, errors.Wrapf(ErrInvalidTransient, "empty token request")
	}

	// request anchor
	requestAnchor := tx.ID()

	// rws
	rws, err := tx.RWSet()
	if err != nil {
		return nil, errors.Wrapf(errors.Join(ErrInvalidProposal, err), "failed to get rws for tx [%s]", tx.ID())
	}
	defer func() {
		// if an error occurred, then call Done on the rwset
		if rws != nil {
			rws.Done()
		}
	}()

	// the rws must be empty
	if len(rws.Namespaces()) != 0 {
		return nil, errors.Wrapf(ErrInvalidProposal, "non empty namespaces")
	}

	if name, version := tx.Chaincode(); name != tmsID.Namespace || version != ChaincodeVersion {
		return nil, errors.Wrapf(ErrInvalidProposal, "invalid chaincode")
	}
	fn, parms := tx.FunctionAndParameters()
	if len(parms) != 0 {
		return nil, errors.Wrapf(ErrInvalidProposal, "invalid parameters")
	}
	if fn != InvokeFunction {
		return nil, errors.Wrapf(ErrInvalidProposal, "invalid function [%s]", fn)
	}

	// copy rws to make sure Done is not invoked on it, see defer above
	returnRws := rws
	rws = nil

	return &Request{
		Tx:               tx,
		Rws:              returnRws,
		TMSID:            tmsID,
		RequestRaw:       requestRaw,
		Anchor:           requestAnchor,
		Tms:              tms,
		PublicParamsHash: tms.PublicParametersManager().PublicParamsHash(),
	}, nil
}

func (r *RequestApprovalResponderView) validateProposal(ctx view.Context, request *Request) error {
	logger.DebugfContext(ctx.Context(), "Validate proposal for TX [%s]", request.Anchor)

	// Get the signed proposal from the underlying Fabric transaction
	signedProposal := request.Tx.Transaction.SignedProposal()
	if signedProposal == nil {
		return errors.Errorf("signed proposal is nil for tx [%s]", request.Anchor)
	}

	// Get the proposal
	proposal := request.Tx.Transaction.Proposal()
	if proposal == nil {
		return errors.Errorf("proposal is nil for tx [%s]", request.Anchor)
	}

	// Verify the proposal signature
	// The signature verification ensures that the proposal was signed by the creator
	creator := request.Tx.Transaction.Creator()
	if len(creator) == 0 {
		return errors.Errorf("creator is empty for tx [%s]", request.Anchor)
	}

	// Get the proposal bytes for signature verification from the signed proposal
	proposalBytes := signedProposal.ProposalBytes()
	if len(proposalBytes) == 0 {
		return errors.Errorf("proposal bytes are empty for tx [%s]", request.Anchor)
	}

	// Verify the signature on the proposal
	signature := signedProposal.Signature()
	if len(signature) == 0 {
		return errors.Errorf("proposal signature is empty for tx [%s]", request.Anchor)
	}

	// Verify the signature over the proposal using the TMS signature service
	// Try to get a verifier for the creator identity - try owner, issuer, and auditor verifiers
	sigService := request.Tms.SigService()

	// Try owner verifier first
	verifier, err := sigService.OwnerVerifier(ctx.Context(), creator)
	if err != nil {
		// Try issuer verifier
		verifier, err = sigService.IssuerVerifier(ctx.Context(), creator)
		if err != nil {
			// Try auditor verifier
			verifier, err = sigService.AuditorVerifier(ctx.Context(), creator)
			if err != nil {
				return errors.Wrapf(err, "failed to get verifier for creator for tx [%s]", request.Anchor)
			}
		}
	}

	// Verify the signature
	err = verifier.Verify(proposalBytes, signature)
	if err != nil {
		return errors.Wrapf(err, "failed to verify proposal signature for tx [%s]", request.Anchor)
	}

	// Validate that the token actions in the request are consistent with the proposal
	// The token request should match what's in the transient data
	// This ensures the relationship between the action, read-write set, and token actions
	if len(request.RequestRaw) == 0 {
		return errors.Errorf("token request is empty for tx [%s]", request.Anchor)
	}

	// The actions will be validated in the validate() method which checks:
	// - Token actions are valid
	// - Read-write set is consistent with the actions
	// - Signatures on token actions are valid

	logger.DebugfContext(ctx.Context(), "Proposal signature verified successfully for TX [%s]", request.Anchor)

	return nil
}

func (r *RequestApprovalResponderView) translate(ctx context.Context, request *Request) error {
	// prepare the rws as usual
	txID := request.Anchor
	w, err := r.getTranslator(txID, request.TMSID.Namespace, request.Rws)
	if err != nil {
		return errors.Wrapf(err, "failed to get translator for tx [%s]", request.Anchor)
	}
	for _, action := range request.Actions {
		if err := w.Write(ctx, action); err != nil {
			return errors.Wrapf(err, "failed to write token action for tx [%s]", txID)
		}
	}
	err = w.AddPublicParamsDependency()
	if err != nil {
		return errors.Wrapf(err, "failed to add public params dependency")
	}
	_, err = w.CommitTokenRequest(request.Meta[common.TokenRequestToSign], true)
	if err != nil {
		return errors.Wrapf(err, "failed to write token request")
	}

	return nil
}

func (r *RequestApprovalResponderView) validate(context view.Context, request *Request, getState tdriver.GetStateFnc) error {
	logger.DebugfContext(context.Context(), "Validate TX [%s]", request.Anchor)

	defer logger.DebugfContext(context.Context(), "Finished validation of TX [%s]", request.Anchor)
	logger.DebugfContext(context.Context(), "Get validator for TX [%s]", request.Anchor)
	validator, err := request.Tms.Validator()
	if err != nil {
		return errors.WithMessagef(err, "failed to get validator [%s]", request.TMSID)
	}
	logger.DebugfContext(context.Context(), "Unmarshal and verify with metadata for TX [%s]", request.Anchor)
	actions, meta, err := validator.UnmarshallAndVerifyWithMetadata(
		context.Context(),
		token2.NewLedgerFromGetter(getState),
		token2.RequestAnchor(request.Anchor),
		request.RequestRaw,
	)
	if err != nil {
		return errors.WithMessagef(err, "failed to verify token request for [%s]", request.Anchor)
	}
	db, err := r.storageProvider.GetStorage(request.TMSID)
	if err != nil {
		return errors.WithMessagef(err, "failed to retrieve db [%s]", request.TMSID)
	}
	logger.DebugfContext(context.Context(), "Append validation record for TX [%s]", request.Anchor)
	if err := db.AppendValidationRecord(
		context.Context(),
		request.Anchor,
		request.RequestRaw,
		meta,
		request.PublicParamsHash,
	); err != nil {
		return errors.WithMessagef(err, "failed to append metadata for [%s]", request.Anchor)
	}
	request.Actions = actions
	request.Meta = meta

	return nil
}

func (r *RequestApprovalResponderView) endorse(ctx view.Context, request *Request) (any, error) {
	// endorse
	logger.DebugfContext(ctx.Context(), "Endorse TX [%s]", request.Anchor)
	endorserID, err := r.endorserService.EndorserID(request.TMSID)
	if err != nil {
		return nil, err
	}

	// write actions into the transaction
	logger.DebugfContext(ctx.Context(), "Translate TX [%s]", request.Anchor)
	err = r.translate(ctx.Context(), request)
	if err != nil {
		return nil, err
	}

	logger.DebugfContext(ctx.Context(), "Endorse proposal for TX [%s]", request.Anchor)
	endorsementResult, err := r.endorserService.Endorse(ctx, request.Tx, endorserID)
	if err != nil {
		logger.Errorf("failed to respond to endorsement [%s]", err)
	}
	logger.DebugfContext(ctx.Context(), "Finished endorsement on TX [%s]", request.Anchor)

	return endorsementResult, err
}
