/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package orion

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/orion"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	session2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/session"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/pkg/errors"
)

type TxStatusRequest struct {
	Network string
	TxID    string
}

type TxStatusResponse struct {
	Status driver.ValidationCode
}

type RequestTxStatusView struct {
	Network driver.Network
	TxID    string
}

func NewRequestTxStatusView(network driver.Network, txID string) *RequestTxStatusView {
	return &RequestTxStatusView{Network: network, TxID: txID}
}

func (r *RequestTxStatusView) Call(context view.Context) (interface{}, error) {
	custodian, err := GetCustodian(view2.GetConfigService(context), r.Network.Name())
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
		Network: r.Network.Name(),
		TxID:    r.TxID,
	}
	if err := session.Send(request); err != nil {
		return nil, errors.Wrapf(err, "failed to send request to custodian [%s]", custodian)
	}
	response := &TxStatusResponse{}
	if err := session.Receive(response); err != nil {
		return nil, errors.Wrapf(err, "failed to receive response from custodian [%s]", custodian)
	}
	return response.Status, nil
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

	status, err := r.process(context, request)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to process request")
	}
	if err := session.Send(&TxStatusResponse{Status: status}); err != nil {
		return nil, errors.Wrapf(err, "failed to send response")
	}
	return nil, nil
}

func (r *RequestTxStatusResponderView) process(context view.Context, request *TxStatusRequest) (driver.ValidationCode, error) {
	ons := orion.GetOrionNetworkService(context, request.Network)
	if ons == nil {
		return driver.Unknown, errors.Errorf("failed to get orion network service for network [%s]", request.Network)
	}
	custodianID, err := GetCustodian(view2.GetConfigService(context), request.Network)
	if err != nil {
		return driver.Unknown, errors.Wrap(err, "failed to get custodian identifier")
	}
	logger.Debugf("open session to orion [%s]", custodianID)
	oSession, err := ons.SessionManager().NewSession(custodianID)
	if err != nil {
		return driver.Unknown, errors.Wrapf(err, "failed to create session to orion network [%s]", request.Network)
	}
	ledger, err := oSession.Ledger()
	if err != nil {
		return driver.Unknown, errors.Wrapf(err, "failed to get ledger for orion network [%s]", request.Network)
	}
	tx, err := ledger.GetTransactionByID(request.TxID)
	if err != nil {
		return driver.Unknown, errors.Wrapf(err, "failed to get transaction [%s] for orion network [%s]", request.TxID, request.Network)
	}

	switch tx.ValidationCode() {
	case orion.VALID:
		return driver.Valid, nil
	default:
		return driver.Invalid, nil
	}
}
