/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package orion

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/orion"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	session2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/session"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/translator"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("token-sdk.network.orion.custodian")

type PublicParamsRequest struct {
	Network   string
	Namespace string
}

type PublicParamsResponse struct {
	Raw []byte
}

type PublicParamsRequestView struct {
	Network   string
	Namespace string
}

func NewPublicParamsRequestView(network string, namespace string) *PublicParamsRequestView {
	return &PublicParamsRequestView{Network: network, Namespace: namespace}
}

func (v *PublicParamsRequestView) Call(context view.Context) (interface{}, error) {
	cp := view2.GetConfigService(context)
	isCustodian, err := IsCustodian(cp)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to check if custodian")
	}
	if isCustodian {
		logger.Debugf("I'm a custodian, connect directly to orion")
		// I'm a custodian, connect directly to orion
		return ReadPublicParameters(context, v.Network, v.Namespace)
	}

	// this is not a custodian, connect to it
	logger.Debugf("I'm not a custodian, connect to custodian")
	custodian, err := GetCustodian(cp, v.Network)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get custodian identifier")
	}
	logger.Debugf("custodian: %s", custodian)
	session, err := session2.NewJSON(context, context.Initiator(), view2.GetIdentityProvider(context).Identity(custodian))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get session to custodian [%s]", custodian)
	}
	request := &PublicParamsRequest{
		Network:   v.Network,
		Namespace: v.Namespace,
	}
	if err := session.Send(request); err != nil {
		return nil, errors.Wrapf(err, "failed to send request to custodian [%s]", custodian)
	}
	response := &PublicParamsResponse{}
	if err := session.Receive(response); err != nil {
		return nil, errors.Wrapf(err, "failed to receive response from custodian [%s]", custodian)
	}
	return response.Raw, nil
}

type RespondPublicParamsRequestView struct{}

func (v *RespondPublicParamsRequestView) Call(context view.Context) (interface{}, error) {
	// receive request
	session := session2.JSON(context)
	request := &PublicParamsRequest{}
	if err := session.Receive(request); err != nil {
		return nil, errors.Wrapf(err, "failed to receive request")
	}
	logger.Debugf("request: %+v", request)

	// Get the public parameters from request network and namespace
	ppRaw, err := ReadPublicParameters(context, request.Network, request.Namespace)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get public parameters from orion network [%s]", request.Network)
	}
	logger.Debugf("reading public parameters, done: %d", len(ppRaw))
	if err := session.Send(&PublicParamsResponse{Raw: ppRaw}); err != nil {
		return nil, errors.Wrapf(err, "failed to send response")
	}
	return nil, nil
}

type AllIssuersValid struct{}

func (i *AllIssuersValid) Validate(creator view.Identity, tokenType string) error {
	return nil
}

func ReadPublicParameters(context view2.ServiceProvider, network, namespace string) ([]byte, error) {
	ons := orion.GetOrionNetworkService(context, network)
	if ons == nil {
		return nil, errors.Errorf("failed to get orion network service for network [%s]", network)
	}
	custodianID, err := GetCustodian(view2.GetConfigService(context), network)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get custodian identifier")
	}
	logger.Debugf("open session to orion [%s]", custodianID)
	oSession, err := ons.SessionManager().NewSession(custodianID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create session to orion network [%s]", network)
	}
	qe, err := oSession.QueryExecutor(namespace)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get query executor for orion network [%s]", network)
	}
	rwset := &ReadOnlyRWSWrapper{qe: qe}
	issuingValidator := &AllIssuersValid{}
	w := translator.New(issuingValidator, "", rwset, "")
	ppRaw, err := w.ReadSetupParameters()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to retrieve public parameters")
	}
	if len(ppRaw) == 0 {
		return nil, errors.Errorf("public parameters are not initiliazed yet")
	}
	logger.Debugf("public parameters read: %d", len(ppRaw))
	return ppRaw, nil
}
