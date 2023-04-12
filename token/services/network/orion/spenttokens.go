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
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/pkg/errors"
)

type SpentTokensRequest struct {
	Network   string
	Namespace string
	IDs       []string
}

type SpentTokensResponse struct {
	Flags []bool
}

type RequestSpentTokensView struct {
	Network   driver.Network
	Namespace string
	IDs       []string
}

func NewRequestSpentTokensView(network driver.Network, namespace string, IDs []string) *RequestSpentTokensView {
	return &RequestSpentTokensView{Network: network, Namespace: namespace, IDs: IDs}
}

func (r *RequestSpentTokensView) Call(context view.Context) (interface{}, error) {
	custodian, err := GetCustodian(view2.GetConfigService(context), r.Network.Name())
	if err != nil {
		return nil, errors.Wrap(err, "failed to get custodian identifier")
	}
	logger.Debugf("custodian: %s", custodian)
	session, err := session2.NewJSON(context, context.Initiator(), view2.GetIdentityProvider(context).Identity(custodian))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get session to custodian [%s]", custodian)
	}
	// TODO: Should we sign the SpentTokens request?
	request := &SpentTokensRequest{
		Network:   r.Network.Name(),
		Namespace: r.Namespace,
		IDs:       r.IDs,
	}
	if err := session.Send(request); err != nil {
		return nil, errors.Wrapf(err, "failed to send request to custodian [%s]", custodian)
	}
	response := &SpentTokensResponse{}
	if err := session.Receive(response); err != nil {
		return nil, errors.Wrapf(err, "failed to receive response from custodian [%s]", custodian)
	}
	return response.Flags, nil
}

type RequestSpentTokensResponderView struct{}

func (r *RequestSpentTokensResponderView) Call(context view.Context) (interface{}, error) {
	// receive request
	session := session2.JSON(context)
	request := &SpentTokensRequest{}
	if err := session.Receive(request); err != nil {
		return nil, errors.Wrapf(err, "failed to receive request")
	}
	logger.Debugf("request: %+v", request)

	flags, err := r.process(context, request)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to process request")
	}
	if err := session.Send(&SpentTokensResponse{Flags: flags}); err != nil {
		return nil, errors.Wrapf(err, "failed to send response")
	}
	return nil, nil
}

func (r *RequestSpentTokensResponderView) process(context view.Context, request *SpentTokensRequest) ([]bool, error) {
	ons := orion.GetOrionNetworkService(context, request.Network)
	if ons == nil {
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
	qe, err := oSession.QueryExecutor(request.Namespace)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get query executor for orion network [%s:%s]", request.Network, request.Namespace)
	}

	tms := token.GetManagementService(context, token.WithTMS(request.Network, "", request.Namespace))
	if tms == nil {
		return nil, errors.Errorf("cannot find tms for [%s:%s]", request.Network, request.Namespace)
	}

	flags := make([]bool, len(request.IDs))
	if tms.PublicParametersManager().PublicParameters().GraphHiding() {
		for i, id := range request.IDs {
			oID := orionKey(id)
			v, err := qe.Get(oID)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get serial number %s", id)
			}
			flags[i] = len(v) != 0
		}
	} else {
		for i, id := range request.IDs {
			oID := orionKey(id)
			v, err := qe.Get(oID)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get serial number %s", id)
			}
			flags[i] = len(v) == 0
		}
	}
	return flags, nil
}
