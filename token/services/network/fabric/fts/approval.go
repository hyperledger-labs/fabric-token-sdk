/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fts

import (
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/vault"

	fabric2 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/services/endorser"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/translator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	"github.com/pkg/errors"
)

const (
	InvokeFunction = "invoke"
)

var logger = logging.MustGetLogger("token-sdk.network.fabric.fts")

type RequestApprovalView struct {
	TMS        *token2.ManagementService
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
	_, tx, err := endorser.NewTransaction(
		context,
		fabric2.WithCreator(r.TxID.Creator),
		fabric2.WithNonce(r.TxID.Nonce),
	)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to create endorser transaction")
	}

	tx.SetProposal(r.TMS.Namespace(), "", InvokeFunction)
	if err := tx.EndorseProposal(); err != nil {
		return nil, errors.WithMessagef(err, "failed to endorse proposal")
	}
	if err := tx.SetTransientState("tmsID", r.TMS.ID()); err != nil {
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

	logger.Debugf("Request Endorsement on tx [%s] to [%v]...", tx.ID(), r.Endorsers)
	_, err = context.RunView(endorser.NewParallelCollectEndorsementsOnProposalView(
		tx,
		r.Endorsers...,
	).WithTimeout(2 * time.Minute))
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to collect endorsements")
	}
	logger.Debugf("Request Endorsement on tx [%s] to [%v]...done", tx.ID(), r.Endorsers)

	rws, err := tx.RWSet()
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get rws")
	}
	rws.Done()
	logger.Debugf("[%s] found [%d] nss [%v]", tx.ID(), len(rws.Namespaces()), rws.Namespaces())

	// Return envelope
	return tx.Envelope()
}

type RequestApprovalResponderView struct{}

func (r *RequestApprovalResponderView) Call(context view.Context) (interface{}, error) {
	// When the borrower runs the CollectEndorsementsView, at some point, the borrower sends the assembled transaction
	// to the approver. Therefore, the approver waits to receive the transaction.
	tx, err := endorser.ReceiveTransaction(context)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to received transaction for approval")
	}
	raw, err := tx.Bytes()
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to marshal transaction [%s]", tx.ID())
	}

	logger.Debugf("Respond to request of approval for tx [%s][%s]", tx.ID(), hash.Hashable(raw).String())

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

	logger.Debugf("evaluate token request on TMS [%s]", tmsID)
	tms := token2.GetManagementService(context, token2.WithTMSID(tmsID))
	if tms == nil {
		return nil, errors.Errorf("cannot find TMS for [%s]", tmsID)
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
	v, err := network.GetInstance(context, tx.Network(), tx.Channel()).Vault(tms.Namespace())
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get vault")
	}
	actions, validationMetadata, err := r.validate(context, tms, tx, requestAnchor, requestRaw, v)
	if err != nil {
		return nil, err
	}

	// endorse
	endorserID, err := r.endorserID(tms, fns)
	if err != nil {
		return nil, err
	}

	// write actions into the transaction
	err = r.translate(tms, tx, validationMetadata, rws, actions...)
	if err != nil {
		return nil, err
	}

	endorsementResult, err := context.RunView(endorser.NewEndorsementOnProposalResponderView(tx, endorserID))
	if err != nil {
		logger.Errorf("failed to respond to endorsement [%s]", err)
	}
	return endorsementResult, err
}

func (r *RequestApprovalResponderView) translate(
	tms *token2.ManagementService,
	tx *endorser.Transaction,
	validationMetadata map[string][]byte,
	rws *fabric2.RWSet,
	actions ...any,
) error {
	// prepare the rws as usual
	w := translator.New(tx.ID(), &rwsWrapper{stub: rws}, tms.Namespace())
	for _, action := range actions {
		if err := w.Write(action); err != nil {
			return errors.Wrapf(err, "failed to write token action for tx [%s]", tx.ID())
		}
	}
	err := w.AddPublicParamsDependency()
	if err != nil {
		return errors.Wrapf(err, "failed to add public params dependency")
	}
	_, err = w.CommitTokenRequest(validationMetadata[vault.TokenRequestToSign], true)
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
	vault *network.Vault,
) ([]any, map[string][]byte, error) {
	validator, err := tms.Validator()
	if err != nil {
		return nil, nil, errors.WithMessagef(err, "failed to get validator [%s:%s]", tms.Network(), tms.Channel())
	}
	qe, err := vault.NewQueryExecutor()
	if err != nil {
		return nil, nil, errors.WithMessagef(err, "failed to get query executor")
	}
	defer qe.Done()
	actions, meta, err := validator.UnmarshallAndVerifyWithMetadata(context.Context(), qe, anchor, requestRaw)
	if err != nil {
		return nil, nil, errors.WithMessagef(err, "failed to verify token request for [%s]", tx.ID())
	}
	db, err := ttxdb.GetByTMSId(context, tms.ID())
	if err != nil {
		return nil, nil, errors.WithMessagef(err, "failed to retrieve db [%s]", tms.ID())
	}
	if err := db.AppendValidationRecord(tx.ID(), requestRaw, meta); err != nil {
		return nil, nil, errors.WithMessagef(err, "failed to append metadata for [%s]", tx.ID())
	}
	return actions, meta, nil
}

func (r *RequestApprovalResponderView) endorserID(tms *token2.ManagementService, fns *fabric2.NetworkService) (view.Identity, error) {
	var endorserIDLabel string
	if err := tms.Configuration().UnmarshalKey("services.network.fabric.fsc_endorsement.id", &endorserIDLabel); err != nil {
		return nil, errors.WithMessage(err, "failed to load endorserID")
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

type rwsWrapper struct {
	stub *fabric2.RWSet
}

func (rwset *rwsWrapper) SetState(namespace string, key string, value []byte) error {
	return rwset.stub.SetState(namespace, key, value)
}

func (rwset *rwsWrapper) GetState(namespace string, key string) ([]byte, error) {
	return rwset.stub.GetState(namespace, key)
}

func (rwset *rwsWrapper) DeleteState(namespace string, key string) error {
	return rwset.stub.DeleteState(namespace, key)
}

func (rwset *rwsWrapper) Bytes() ([]byte, error) {
	return nil, nil
}

func (rwset *rwsWrapper) Done() {
}
