/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package orion

import (
	errors2 "errors"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/translator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	session2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/json/session"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
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
	span := trace.SpanFromContext(context.Context())

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
	dbManager     *DBManager
	statusCache   TxStatusResponseCache
	keyTranslator translator.KeyTranslator
}

func (r *RequestTxStatusResponderView) Call(context view.Context) (interface{}, error) {
	span := trace.SpanFromContext(context.Context())

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
		if err2 := session.SendError(err.Error()); err2 != nil {
			return nil, errors.Wrapf(errors2.Join(err, err2), "failed to process request")
		}
		return nil, errors.Wrapf(err, "failed to process request")
	}
	if response.Status == driver.Valid && len(response.TokenRequestReference) == 0 {
		panic("invalid result for [" + request.TxID + "]")
	}
	span.AddEvent("send_tx_status_response")
	logger.Debugf("send tx status response for [%s]: [%d]", request.TxID, response.Status)
	if err := session.SendWithContext(context.Context(), response); err != nil {
		return nil, errors.Wrapf(err, "failed to send response")
	}
	return nil, nil
}

func (r *RequestTxStatusResponderView) process(context view.Context, request *TxStatusRequest) (*TxStatusResponse, error) {
	if status, ok := r.statusCache.Get(request.TxID); ok && status.Status != driver.Busy {
		if status.Status != driver.Valid || len(status.TokenRequestReference) != 0 {
			return status, nil
		}
	}
	if status, err := NewStatusFetcher(r.dbManager, r.keyTranslator).FetchStatus(request.Network, request.Namespace, request.TxID); err == nil {
		r.statusCache.Add(request.TxID, status)
		return status, nil
	} else {
		return nil, err
	}
}
