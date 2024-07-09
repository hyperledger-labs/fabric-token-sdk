/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package orion

import (
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/orion"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	session2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/session"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/translator"
	"github.com/pkg/errors"
)

type ApprovalRequest struct {
	Network   string
	Namespace string
	TxID      string
	Request   []byte
}

type ApprovalResponse struct {
	Envelope []byte
}

type RequestApprovalView struct {
	Network    *orion.NetworkService
	Namespace  string
	RequestRaw []byte
	Signer     view.Identity
	TxID       string
}

func NewRequestApprovalView(network *orion.NetworkService, namespace string, requestRaw []byte, signer view.Identity, txID string) *RequestApprovalView {
	return &RequestApprovalView{Network: network, Namespace: namespace, RequestRaw: requestRaw, Signer: signer, TxID: txID}
}

func (r *RequestApprovalView) Call(context view.Context) (interface{}, error) {
	custodian, err := GetCustodian(view2.GetConfigService(context), r.Network.Name())
	if err != nil {
		return nil, errors.Wrap(err, "failed to get custodian identifier")
	}
	logger.Debugf("custodian: %s", custodian)
	session, err := session2.NewJSON(context, context.Initiator(), view2.GetIdentityProvider(context).Identity(custodian))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get session to custodian [%s]", custodian)
	}
	// TODO: Should we sign the approval request?
	request := &ApprovalRequest{
		Network:   r.Network.Name(),
		Namespace: r.Namespace,
		TxID:      r.TxID,
		Request:   r.RequestRaw,
	}
	if err := session.Send(request); err != nil {
		return nil, errors.Wrapf(err, "failed to send request to custodian [%s]", custodian)
	}
	response := &ApprovalResponse{}
	if err := session.ReceiveWithTimeout(response, 30*time.Second); err != nil {
		return nil, errors.Wrapf(err, "failed to receive response from custodian [%s]", custodian)
	}
	env := r.Network.TransactionManager().NewEnvelope()
	if err := env.FromBytes(response.Envelope); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal transaction")
	}
	return env, nil
}

type RequestApprovalResponderView struct {
	dbManager *DBManager
}

func (r *RequestApprovalResponderView) Call(context view.Context) (interface{}, error) {
	// receive request
	session := session2.JSON(context)
	request := &ApprovalRequest{}
	if err := session.Receive(request); err != nil {
		return nil, errors.Wrapf(err, "failed to receive request")
	}
	logger.Debugf("request: %+v", request)

	txRaw, err := r.process(context, request)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to process request")
	}
	if err := session.Send(&ApprovalResponse{Envelope: txRaw}); err != nil {
		return nil, errors.Wrapf(err, "failed to send response")
	}
	return nil, nil
}

func (r *RequestApprovalResponderView) process(context view.Context, request *ApprovalRequest) ([]byte, error) {
	ppRaw, err := ReadPublicParameters(context, request.Network, request.Namespace)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read public parameters")
	}
	ds, err := driver.GetTokenDriverService(context)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get token driver")
	}
	pp, err := ds.PublicParametersFromBytes(ppRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal public parameters")
	}
	validator, err := ds.NewValidator(token.TMSID{Network: request.Network, Namespace: request.Namespace}, pp)

	if err != nil {
		return nil, errors.Wrapf(err, "failed to create validator")
	}

	// commit
	for i := 0; i < 3; i++ {
		envelopeRaw, err := r.commit(context, request, validator)
		if err != nil {
			logger.Errorf("failed to commit transaction [%s], retry [%d]", err, i)
			time.Sleep(100 * time.Minute)
			continue
		}
		return envelopeRaw, nil
	}

	return nil, errors.Wrapf(err, "failed to commit transaction [%s]", request.TxID)
}

func (r *RequestApprovalResponderView) commit(context view.Context, request *ApprovalRequest, validator driver.Validator) ([]byte, error) {
	sm, err := r.dbManager.GetSessionManager(request.Network)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get session manager for network [%s]", request.Network)
	}
	oSession, err := sm.GetSession()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create session to orion network [%s]", request.Network)
	}
	qe, err := oSession.QueryExecutor(request.Namespace)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get query executor for orion network [%s]", request.Network)
	}
	actions, attributes, err := token.NewValidator(validator).UnmarshallAndVerifyWithMetadata(
		context.Context(),
		&LedgerWrapper{qe: qe},
		request.TxID,
		request.Request,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshall and verify request")
	}

	// Write
	tx, err := sm.Orion.TransactionManager().NewTransaction(request.TxID, sm.CustodianID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create transaction [%s]", request.TxID)
	}
	rws := &TxRWSWrapper{
		me: sm.CustodianID,
		db: request.Namespace,
		tx: tx,
	}
	t := translator.New(request.TxID, rws, "")
	for _, action := range actions {
		err = t.Write(action)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to write action")
		}
	}
	err = t.CommitTokenRequest(attributes[common.TokenRequestToSign], true)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to commit token request")
	}

	// close transaction
	envelopeRaw, err := tx.SignAndClose()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to sign and close transaction [%s]", request.TxID)
	}
	return envelopeRaw, nil
}

type LedgerWrapper struct {
	qe *orion.SessionQueryExecutor
}

func (l *LedgerWrapper) GetState(key string) ([]byte, error) {
	return l.qe.Get(orionKey(key))
}
