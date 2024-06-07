/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package orion

import (
	"encoding/base64"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/keys"

	errors2 "github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/orion"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	session2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/session"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger-labs/orion-sdk-go/pkg/bcdb"
	"github.com/pkg/errors"
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
}

func NewRequestTxStatusView(network string, namespace string, txID string) *RequestTxStatusView {
	return &RequestTxStatusView{Network: network, Namespace: namespace, TxID: txID}
}

func (r *RequestTxStatusView) Call(context view.Context) (interface{}, error) {
	custodian, err := GetCustodian(view2.GetConfigService(context), r.Network)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get custodian identifier")
	}
	logger.Debugf("custodian: %s", custodian)
	session, err := session2.NewJSON(context, context.Initiator(), view2.GetIdentityProvider(context).Identity(custodian))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get session to custodian [%s]", custodian)
	}
	// TODO: Should we sign the txStatus request?
	request := &TxStatusRequest{
		Network:   r.Network,
		Namespace: r.Namespace,
		TxID:      r.TxID,
	}
	if err := session.Send(request); err != nil {
		return nil, errors.Wrapf(err, "failed to send request to custodian [%s]", custodian)
	}
	response := &TxStatusResponse{}
	if err := session.Receive(response); err != nil {
		return nil, errors.Wrapf(err, "failed to receive response from custodian [%s]", custodian)
	}
	return response, nil
}

type RequestTxStatusResponderView struct{}

func (r *RequestTxStatusResponderView) Call(context view.Context) (interface{}, error) {
	// receive request
	session := session2.JSON(context)
	request := &TxStatusRequest{}
	if err := session.Receive(request); err != nil {
		return nil, errors.Wrapf(err, "failed to receive request")
	}
	logger.Debugf("request: %+v", request)

	response, err := r.process(context, request)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to process request")
	}
	if err := session.Send(response); err != nil {
		return nil, errors.Wrapf(err, "failed to send response")
	}
	return nil, nil
}

func (r *RequestTxStatusResponderView) process(context view.Context, request *TxStatusRequest) (*TxStatusResponse, error) {
	ons, err := orion.GetOrionNetworkService(context, request.Network)
	if err != nil {
		return nil, errors.Errorf("failed to get orion network service for network [%s]", request.Network)
	}
	custodianID, err := GetCustodian(view2.GetConfigService(context), request.Network)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get custodian identifier")
	}
	logger.Debugf("open session to orion [%s]", custodianID)
	oSession, err := ons.SessionManager().NewSession(custodianID)
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
		logger.Debugf("retrieved token request hash for [%s][%s]:[%s]", key, request.TxID, base64.StdEncoding.EncodeToString(trRef))
	}

	switch tx.ValidationCode() {
	case orion.VALID:
		return &TxStatusResponse{Status: driver.Valid, TokenRequestReference: trRef}, nil
	default:
		return &TxStatusResponse{Status: driver.Invalid, TokenRequestReference: trRef}, nil
	}
}
