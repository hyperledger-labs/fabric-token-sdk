/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fsc

import (
	"context"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	fabric2 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/services/endorser"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/comm/session"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/translator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

const (
	TransientTMSIDKey        = "tmsID"
	TransientTokenRequestKey = "token_request"

	ChaincodeVersion = "1.0"
	InvokeFunction   = "invoke"
)

// Transaction models a transaction that requires endorsement
type Transaction interface {
	ID() string
}

type Request struct {
	Tx         *endorser.Transaction
	Rws        *fabric2.RWSet
	TMSID      token2.TMSID
	Anchor     string
	RequestRaw []byte
	Actions    []interface{}
	Meta       map[string][]byte
	Tms        *token2.ManagementService
}

type Translator interface {
	AddPublicParamsDependency() error
	CommitTokenRequest(raw []byte, storeHash bool) ([]byte, error)
	Write(ctx context.Context, action any) error
}

type TranslatorProviderFunc = func(txID string, namespace string, rws *fabric2.RWSet) (Translator, error)

type RequestApprovalResponderView struct {
	keyTranslator translator.KeyTranslator
	getTranslator TranslatorProviderFunc
}

func NewRequestApprovalResponderView(keyTranslator translator.KeyTranslator, getTranslator TranslatorProviderFunc) *RequestApprovalResponderView {
	return &RequestApprovalResponderView{keyTranslator: keyTranslator, getTranslator: getTranslator}
}

func (r *RequestApprovalResponderView) Call(context view.Context) (interface{}, error) {
	// receive
	request, err := r.receive(context)
	if err != nil {
		return nil, err
	}
	defer request.Rws.Done()

	// validate
	err = r.validate(context, request, func(id token.ID) ([]byte, error) {
		key, err := r.keyTranslator.CreateOutputKey(id.TxId, id.Index)
		if err != nil {
			return nil, errors.WithMessagef(err, "failed to create token key for id [%s]", id)
		}
		return request.Rws.GetDirectState(request.TMSID.Namespace, key)
	})
	if err != nil {
		return nil, err
	}

	// endorse
	return r.endorse(context, request)
}

func (r *RequestApprovalResponderView) receive(context view.Context) (*Request, error) {
	logger.DebugfContext(context.Context(), "Waiting for transaction on context [%s]", context.ID())
	_, tx, err := endorser.NewTransactionFromBytes(context, session.ReadFirstMessageOrPanic(context))
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to received transaction for approval")
	}
	logger.DebugfContext(context.Context(), "Received transaction [%s] for endorsement on context [%s]", tx.ID(), context.ID())
	defer logger.DebugfContext(context.Context(), "Return endorsement result for TX [%s]", tx.ID())

	// validate transient

	// we expect 3 transient keys
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
	tms, err := token2.GetManagementService(context, token2.WithTMSID(tmsID))
	if err != nil {
		return nil, errors.Wrapf(errors.Join(err, ErrInvalidTransient), "cannot find TMS for [%s]", tmsID)
	}
	if !tms.ID().Equal(tmsID) {
		return nil, errors.Wrapf(errors.Join(err, ErrInvalidTransient), "tms ids do not match")
	}
	logger.DebugfContext(context.Context(), "evaluate token request on TMS [%s]", tmsID)

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
		return nil, errors.WithMessagef(err, "failed to get rws for tx [%s]", tx.ID())
	}
	// the rws must be empty
	if len(rws.Namespaces()) != 0 {
		rws.Done()
		return nil, errors.Wrapf(ErrInvalidTransient, "non empty namespaces")
	}

	// TODO: check that tx contains a valid endorser proposal
	if name, version := tx.Chaincode(); name != tmsID.Namespace || version != ChaincodeVersion {
		return nil, errors.Wrapf(ErrInvalidProposal, "invalid chaincode")
	}
	fn, parms := tx.FunctionAndParameters()
	if len(parms) != 0 {
		return nil, errors.Wrapf(ErrInvalidProposal, "invalid parameters")
	}
	if fn != InvokeFunction {
		return nil, errors.Wrapf(ErrInvalidProposal, "invalid function")
	}

	return &Request{
		Tx:         tx,
		Rws:        rws,
		TMSID:      tmsID,
		RequestRaw: requestRaw,
		Anchor:     requestAnchor,
		Tms:        tms,
	}, nil
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

func (r *RequestApprovalResponderView) validate(context view.Context, request *Request, getState driver2.GetStateFnc) error {
	logger.DebugfContext(context.Context(), "Validate TX [%s]", request.Anchor)
	tms := request.Tms

	defer logger.DebugfContext(context.Context(), "Finished validation of TX [%s]", request.Anchor)
	logger.DebugfContext(context.Context(), "Get validator for TX [%s]", request.Anchor)
	validator, err := tms.Validator()
	if err != nil {
		return errors.WithMessagef(err, "failed to get validator [%s:%s]", tms.Network(), tms.Channel())
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
	db, err := ttxdb.GetByTMSId(context, tms.ID())
	if err != nil {
		return errors.WithMessagef(err, "failed to retrieve db [%s]", tms.ID())
	}
	logger.DebugfContext(context.Context(), "Append validation record for TX [%s]", request.Anchor)
	if err := db.AppendValidationRecord(
		context.Context(),
		request.Anchor,
		request.RequestRaw,
		meta,
		tms.PublicParametersManager().PublicParamsHash(),
	); err != nil {
		return errors.WithMessagef(err, "failed to append metadata for [%s]", request.Anchor)
	}
	request.Actions = actions
	request.Meta = meta
	return nil
}

func (r *RequestApprovalResponderView) endorserID(tms *token2.ManagementService, fns *fabric2.NetworkService) (view.Identity, error) {
	var endorserIDLabel string
	if err := tms.Configuration().UnmarshalKey("services.network.fabric.fsc_endorsement.id", &endorserIDLabel); err != nil {
		return nil, errors.WithMessagef(err, "failed to load endorserID")
	}
	var endorserID view.Identity
	if len(endorserIDLabel) == 0 {
		endorserID = fns.LocalMembership().DefaultIdentity()
	} else {
		var err error
		endorserID, err = fns.LocalMembership().GetIdentityByID(endorserIDLabel)
		if err != nil {
			return nil, errors.WithMessagef(err, "cannot find local endorser identity for [%s]", endorserIDLabel)
		}
	}
	if endorserID.IsNone() {
		return nil, errors.Errorf("cannot find local endorser identity for [%s]", endorserIDLabel)
	}
	if _, err := fns.SignerService().GetSigner(endorserID); err != nil {
		return nil, errors.WithMessagef(err, "cannot find fabric signer for identity [%s:%s]", endorserIDLabel, endorserID)
	}
	return endorserID, nil
}

func (r *RequestApprovalResponderView) endorse(ctx view.Context, request *Request) (any, error) {
	// endorse
	logger.DebugfContext(ctx.Context(), "Endorse TX [%s]", request.Anchor)
	fns, err := fabric2.GetFabricNetworkService(ctx, request.TMSID.Network)
	if err != nil {
		return nil, errors.WithMessagef(err, "cannot find fabric network for [%s]", request.TMSID.Network)
	}
	endorserID, err := r.endorserID(request.Tms, fns)
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
	endorsementResult, err := ctx.RunView(endorser.NewEndorsementOnProposalResponderView(request.Tx, endorserID))
	if err != nil {
		logger.Errorf("failed to respond to endorsement [%s]", err)
	}
	logger.DebugfContext(ctx.Context(), "Finished endorsement on TX [%s]", request.Anchor)
	return endorsementResult, err
}

type RWSWrapper struct {
	Stub *fabric2.RWSet
}

func (rwset *RWSWrapper) SetState(namespace string, key string, value []byte) error {
	return rwset.Stub.SetState(namespace, key, value)
}

func (rwset *RWSWrapper) GetState(namespace string, key string) ([]byte, error) {
	return rwset.Stub.GetState(namespace, key)
}

func (rwset *RWSWrapper) DeleteState(namespace string, key string) error {
	return rwset.Stub.DeleteState(namespace, key)
}
