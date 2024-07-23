/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package orion

import (
	"encoding/base64"

	errors2 "github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/orion"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	session2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/session"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/keys"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger-labs/orion-sdk-go/pkg/bcdb"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

type TxStatusRequest struct {
	Network   string
	Namespace string
	TxID      string
}

type TxStatusResponse struct {
	Status                driver.ValidationCode
	TokenRequestReference []byte
}

type RequestTxStatusView struct {
	Network   string
	Namespace string
	TxID      string
	dbManager *DBManager
}

func NewRequestTxStatusView(network string, namespace string, txID string, dbManager *DBManager) *RequestTxStatusView {
	return &RequestTxStatusView{Network: network, Namespace: namespace, TxID: txID, dbManager: dbManager}
}

func (r *RequestTxStatusView) Call(context view.Context) (interface{}, error) {
	span := context.StartSpan("request_tx_status_view")
	defer span.End()
	sm, err := r.dbManager.GetSessionManager(r.Network)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed getting session manager for network [%s]", r.Network)
	}
	custodian := sm.CustodianID
	session, err := session2.NewJSON(context, context.Initiator(), view2.GetIdentityProvider(context).Identity(custodian))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get session to custodian [%s]", custodian)
	}
	logger.Debugf("request tx status for [%s]", r.TxID)

	// TODO: Should we sign the txStatus request?
	request := &TxStatusRequest{
		Network:   r.Network,
		Namespace: r.Namespace,
		TxID:      r.TxID,
	}
	span.AddEvent("send_tx_status_request")
	if err := session.SendWithContext(context.Context(), request); err != nil {
		return nil, errors.Wrapf(err, "failed to send request to custodian [%s]", custodian)
	}
	response := &TxStatusResponse{}
	span.AddEvent("receive_tx_status_response")
	if err := session.Receive(response); err != nil {
		return nil, errors.Wrapf(err, "failed to receive response from custodian [%s]", custodian)
	}
	logger.Debugf("got tx status response for [%s]: [%d]", r.TxID, response.Status)
	return response, nil
}

type RequestTxStatusResponderView struct {
	dbManager   *DBManager
	statusCache TxStatusResponseCache
}

func (r *RequestTxStatusResponderView) Call(context view.Context) (interface{}, error) {
	span := context.StartSpan("request_tx_status_responder_view")
	defer span.End()
	// receive request
	session := session2.JSON(context)
	request := &TxStatusRequest{}
	span.AddEvent("receive_tx_status_request")
	if err := session.Receive(request); err != nil {
		return nil, errors.Wrapf(err, "failed to receive request")
	}
	logger.Debugf("got tx status request for [%s]: [%+v]", request.TxID, request)

	span.AddEvent("process_tx_status_request")
	response, err := r.process(context, request)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to process request")
	}
	span.AddEvent("send_tx_status_response")
	logger.Debugf("send tx status response for [%s]: [%d]", request.TxID, response.Status)
	if err := session.SendWithContext(context.Context(), response); err != nil {
		return nil, errors.Wrapf(err, "failed to send response")
	}
	return nil, nil
}

func (r *RequestTxStatusResponderView) process(context view.Context, request *TxStatusRequest) (*TxStatusResponse, error) {
	response, ok := r.statusCache.Get(request.TxID)
	if ok && response.Status != driver.Busy {
		return response, nil
	}

	sm, err := r.dbManager.GetSessionManager(request.Network)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get session manager for network [%s]", request.Network)
	}
	oSession, err := sm.GetSession()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create session to orion network [%s]", request.Network)
	}
	ledger, err := oSession.Ledger()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get ledger for orion network [%s]", request.Network)
	}
	tx, err := ledger.GetTransactionByID(request.TxID)
	if err != nil {
		if errors2.HasType(err, &bcdb.ErrorNotFound{}) {
			return &TxStatusResponse{Status: driver.Unknown}, nil
		}
		return nil, errors.Wrapf(err, "failed to get transaction [%s] for orion network [%s]", request.TxID, request.Network)
	}

	var trRef []byte
	if len(request.Namespace) != 0 {
		// fetch token request reference
		qe, err := oSession.QueryExecutor(request.Namespace)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get query executor [%s] for orion network [%s]", request.TxID, request.Network)
		}
		key, err := keys.CreateTokenRequestKey(request.TxID)
		if err != nil {
			return nil, errors.Errorf("can't create for token request '%s'", request.TxID)
		}
		trRef, err = qe.Get(orionKey(key))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get token request reference [%s] for orion network [%s]", request.TxID, request.Network)
		}
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("retrieved token request hash for [%s][%s]:[%s]", key, request.TxID, base64.StdEncoding.EncodeToString(trRef))
		}
	}

	if response == nil {
		response = &TxStatusResponse{
			TokenRequestReference: trRef,
		}
	}
	switch tx.ValidationCode() {
	case orion.VALID:
		response.Status = driver.Valid
	default:
		response.Status = driver.Invalid
	}
	r.statusCache.Add(request.TxID, response)
	return response, nil
}
