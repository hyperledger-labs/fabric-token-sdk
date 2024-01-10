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
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/rws/keys"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type QueryTokensRequest struct {
	Network   string
	Namespace string
	IDs       []*token2.ID
}

type QueryTokensResponse struct {
	Content [][]byte
}

type RequestQueryTokensView struct {
	Network   driver.Network
	Namespace string
	IDs       []*token2.ID
}

func NewRequestQueryTokensView(network driver.Network, namespace string, IDs []*token2.ID) *RequestQueryTokensView {
	return &RequestQueryTokensView{Network: network, Namespace: namespace, IDs: IDs}
}

func (r *RequestQueryTokensView) Call(context view.Context) (interface{}, error) {
	custodian, err := GetCustodian(view2.GetConfigService(context), r.Network.Name())
	if err != nil {
		return nil, errors.Wrap(err, "failed to get custodian identifier")
	}
	logger.Debugf("custodian: %s", custodian)
	session, err := session2.NewJSON(context, context.Initiator(), view2.GetIdentityProvider(context).Identity(custodian))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get session to custodian [%s]", custodian)
	}
	// TODO: Should we sign the QueryTokens request?
	request := &QueryTokensRequest{
		Network:   r.Network.Name(),
		Namespace: r.Namespace,
		IDs:       r.IDs,
	}
	if err := session.Send(request); err != nil {
		return nil, errors.Wrapf(err, "failed to send request to custodian [%s]", custodian)
	}
	response := &QueryTokensResponse{}
	if err := session.Receive(response); err != nil {
		return nil, errors.Wrapf(err, "failed to receive response from custodian [%s]", custodian)
	}
	return response.Content, nil
}

type RequestQueryTokensResponderView struct{}

func (r *RequestQueryTokensResponderView) Call(context view.Context) (interface{}, error) {
	// receive request
	session := session2.JSON(context)
	request := &QueryTokensRequest{}
	if err := session.Receive(request); err != nil {
		return nil, errors.Wrapf(err, "failed to receive request")
	}
	logger.Debugf("request: %+v", request)

	tokenContent, err := r.process(context, request)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to process request")
	}
	if err := session.Send(&QueryTokensResponse{Content: tokenContent}); err != nil {
		return nil, errors.Wrapf(err, "failed to send response")
	}
	return nil, nil
}

func (r *RequestQueryTokensResponderView) process(context view.Context, request *QueryTokensRequest) ([][]byte, error) {
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

	var res [][]byte
	var errs []error
	for _, id := range request.IDs {
		outputID, err := keys.CreateTokenKey(id.TxId, id.Index)
		if err != nil {
			errs = append(errs, errors.Errorf("error creating output ID: %s", err))
			continue
		}
		logger.Debugf("query state [%s:%s]", id, outputID)
		bytes, err := qe.Get(orionKey(outputID))
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "failed getting output for [%s]", outputID))
			// return nil, errors.Wrapf(err, "failed getting output for [%s]", outputID)
			continue
		}
		if len(bytes) == 0 {
			errs = append(errs, errors.Errorf("output for key [%s] does not exist", outputID))
			continue
		}
		res = append(res, bytes)
	}
	if len(errs) != 0 {
		return nil, errors.Errorf("failed quering tokens [%v] with errs [%d][%v]", request.IDs, len(errs), errs)
	}
	return res, nil
}
