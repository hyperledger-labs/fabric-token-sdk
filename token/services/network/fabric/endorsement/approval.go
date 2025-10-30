/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package endorsement

import (
	"context"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	fabric2 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/services/endorser"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	driver2 "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/translator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type RequestApprovalView struct {
	TMSID      token2.TMSID
	TxID       driver.TxID
	RequestRaw []byte
	// RequestAnchor, if not nil it will instruct the approver to verify the token request using this anchor and not the transaction it.
	// This is to be used only for testing.
	RequestAnchor string
	// Nonce, if not nil it will be appended to the messages to sign.
	// This is to be used only for testing.
	Nonce []byte
	// Endorsers are the identities of the FSC node that play the role of endorser
	Endorsers []view.Identity
}

func (r *RequestApprovalView) Call(context view.Context) (interface{}, error) {
	logger.DebugfContext(context.Context(), "request approval...")

	_, tx, err := endorser.NewTransaction(
		context,
		fabric2.WithCreator(r.TxID.Creator),
		fabric2.WithNonce(r.TxID.Nonce),
	)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to create endorser transaction")
	}

	tms, err := token2.GetManagementService(context, token2.WithTMSID(r.TMSID))
	if err != nil {
		return nil, errors.WithMessagef(err, "no token management service for [%s]", r.TMSID)
	}
	tx.SetProposal(tms.Namespace(), "", InvokeFunction)
	if err := tx.EndorseProposal(); err != nil {
		return nil, errors.WithMessagef(err, "failed to endorse proposal")
	}
	if err := tx.SetTransientState("tmsID", tms.ID()); err != nil {
		return nil, errors.WithMessagef(err, "failed to set TMS ID transient")
	}
	if err := tx.SetTransient("token_request", r.RequestRaw); err != nil {
		return nil, errors.WithMessagef(err, "failed to set token request transient")
	}
	if len(r.RequestAnchor) != 0 {
		if err := tx.SetTransient("RequestAnchor", []byte(r.RequestAnchor)); err != nil {
			return nil, errors.WithMessagef(err, "failed to set token request transient")
		}
	}
	if len(r.Nonce) != 0 {
		if err := tx.SetTransient("Nonce", r.Nonce); err != nil {
			return nil, errors.WithMessagef(err, "failed to set token request transient")
		}
	}

	logger.DebugfContext(context.Context(), "request endorsement on tx [%s] to [%v]...", tx.ID(), r.Endorsers)
	_, err = context.RunView(endorser.NewParallelCollectEndorsementsOnProposalView(
		tx,
		r.Endorsers...,
	).WithTimeout(2 * time.Minute))
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to collect endorsements")
	}
	logger.DebugfContext(context.Context(), "request endorsement done")

	// Return envelope
	env, err := tx.Envelope()
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to retrieve envelope for endorsement")
	}
	logger.DebugfContext(context.Context(), "envelope ready")

	return env, nil
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
	// When the borrower runs the CollectEndorsementsView, at some point, the borrower sends the assembled transaction
	// to the approver. Therefore, the approver waits to receive the transaction.
	logger.DebugfContext(context.Context(), "Waiting for transaction on context [%s]", context.ID())
	tx, err := endorser.ReceiveTransaction(context)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to received transaction for approval")
	}
	logger.DebugfContext(context.Context(), "Received transaction [%s] for endorsement on context [%s]", tx.ID(), context.ID())
	defer logger.DebugfContext(context.Context(), "Return endorsement result for TX [%s]", tx.ID())
	raw, err := tx.Bytes()
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to marshal transaction [%s]", tx.ID())
	}

	logger.DebugfContext(context.Context(), "Respond to request of approval for tx [%s][%s]", tx.ID(), utils.Hashable(raw))

	var tmsID token2.TMSID
	if err := tx.GetTransientState("tmsID", &tmsID); err != nil {
		return nil, errors.WithMessagef(err, "failed to get TMS ID from transient [%s]", tx.ID())
	}
	requestRaw := tx.GetTransient("token_request")
	if len(requestRaw) == 0 {
		return nil, errors.Errorf("failed to get token request from transient [%s], it is empty", tx.ID())
	}
	requestAnchor := string(tx.GetTransient("RequestAnchor"))
	if len(requestAnchor) == 0 {
		requestAnchor = tx.ID()
	}

	logger.DebugfContext(context.Context(), "evaluate token request on TMS [%s]", tmsID)
	tms, err := token2.GetManagementService(context, token2.WithTMSID(tmsID))
	if err != nil {
		return nil, errors.WithMessagef(err, "cannot find TMS for [%s]", tmsID)
	}

	rws, err := tx.RWSet()
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get rws for tx [%s]", tx.ID())
	}
	defer rws.Done()

	fns, err := fabric2.GetFabricNetworkService(context, tms.Network())
	if err != nil {
		return nil, errors.WithMessagef(err, "cannot find fabric network for [%s]", tms.Network())
	}

	// validate token request
	logger.DebugfContext(context.Context(), "Validate TX [%s]", tx.ID())
	actions, validationMetadata, err := r.validate(context, tms, tx, requestAnchor, requestRaw, func(id token.ID) ([]byte, error) {
		key, err := r.keyTranslator.CreateOutputKey(id.TxId, id.Index)
		if err != nil {
			return nil, errors.WithMessagef(err, "failed to create token key for id [%s]", id)
		}
		return rws.GetDirectState(tms.Namespace(), key)
	})

	if err != nil {
		return nil, err
	}

	// endorse
	logger.DebugfContext(context.Context(), "Endorse TX [%s]", tx.ID())
	endorserID, err := r.endorserID(tms, fns)
	if err != nil {
		return nil, err
	}

	// write actions into the transaction
	logger.DebugfContext(context.Context(), "Translate TX [%s]", tx.ID())
	err = r.translate(context.Context(), tms, tx, validationMetadata, rws, actions...)
	if err != nil {
		return nil, err
	}

	logger.DebugfContext(context.Context(), "Endorse proposal for TX [%s]", tx.ID())
	endorsementResult, err := context.RunView(endorser.NewEndorsementOnProposalResponderView(tx, endorserID))
	if err != nil {
		logger.Errorf("failed to respond to endorsement [%s]", err)
	}
	logger.DebugfContext(context.Context(), "Finished endorsement on TX [%s]", tx.ID())
	return endorsementResult, err
}

func (r *RequestApprovalResponderView) translate(
	ctx context.Context,
	tms *token2.ManagementService,
	tx *endorser.Transaction,
	validationMetadata map[string][]byte,
	rws *fabric2.RWSet,
	actions ...any,
) error {
	// prepare the rws as usual
	txID := tx.ID()
	w, err := r.getTranslator(txID, tms.Namespace(), rws)
	if err != nil {
		return errors.Wrapf(err, "failed to get translator for tx [%s]", tx.ID())
	}
	for _, action := range actions {
		if err := w.Write(ctx, action); err != nil {
			return errors.Wrapf(err, "failed to write token action for tx [%s]", txID)
		}
	}
	err = w.AddPublicParamsDependency()
	if err != nil {
		return errors.Wrapf(err, "failed to add public params dependency")
	}
	_, err = w.CommitTokenRequest(validationMetadata[common.TokenRequestToSign], true)
	if err != nil {
		return errors.Wrapf(err, "failed to write token request")
	}
	return nil
}

func (r *RequestApprovalResponderView) validate(
	context view.Context,
	tms *token2.ManagementService,
	tx *endorser.Transaction,
	anchor string,
	requestRaw []byte,
	getState driver2.GetStateFnc,
) ([]any, map[string][]byte, error) {
	defer logger.DebugfContext(context.Context(), "Finished validation of TX [%s]", tx.ID())
	logger.DebugfContext(context.Context(), "Get validator for TX [%s]", tx.ID())
	validator, err := tms.Validator()
	if err != nil {
		return nil, nil, errors.WithMessagef(err, "failed to get validator [%s:%s]", tms.Network(), tms.Channel())
	}
	logger.DebugfContext(context.Context(), "Unmarshal and verify with metadata for TX [%s]", tx.ID())
	actions, meta, err := validator.UnmarshallAndVerifyWithMetadata(
		context.Context(),
		token2.NewLedgerFromGetter(getState),
		token2.RequestAnchor(anchor),
		requestRaw,
	)
	if err != nil {
		return nil, nil, errors.WithMessagef(err, "failed to verify token request for [%s]", tx.ID())
	}
	db, err := ttxdb.GetByTMSId(context, tms.ID())
	if err != nil {
		return nil, nil, errors.WithMessagef(err, "failed to retrieve db [%s]", tms.ID())
	}
	logger.DebugfContext(context.Context(), "Append validation record for TX [%s]", tx.ID())
	if err := db.AppendValidationRecord(
		context.Context(),
		tx.ID(),
		requestRaw,
		meta,
		tms.PublicParametersManager().PublicParamsHash(),
	); err != nil {
		return nil, nil, errors.WithMessagef(err, "failed to append metadata for [%s]", tx.ID())
	}
	return actions, meta, nil
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
