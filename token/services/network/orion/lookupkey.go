package orion

import (
	"time"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	session2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/session"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/pkg/errors"
)

type LookupKeyRequest struct {
	Network   string
	Namespace string
}

type LookupKeyResponse struct {
	Raw []byte
}

type LookupKeyRequestView struct {
	Network      string
	Namespace    string
	StartingTxID string
	Key          string
	Timeout      time.Duration
}

func NewLookupKeyRequestView(network string, namespace string, startingTxID string, key string, timeout time.Duration) *LookupKeyRequestView {
	return &LookupKeyRequestView{
		Network:      network,
		Namespace:    namespace,
		StartingTxID: startingTxID,
		Key:          key,
		Timeout:      timeout,
	}
}

func (v *LookupKeyRequestView) Call(context view.Context) (interface{}, error) {
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
	request := &LookupKeyRequest{
		Network:   v.Network,
		Namespace: v.Namespace,
	}
	if err := session.Send(request); err != nil {
		return nil, errors.Wrapf(err, "failed to send request to custodian [%s]", custodian)
	}
	response := &LookupKeyResponse{}
	if err := session.Receive(response); err != nil {
		return nil, errors.Wrapf(err, "failed to receive response from custodian [%s]", custodian)
	}
	return response.Raw, nil
}

type RespondLookupKeyRequestView struct{}

func (v *RespondLookupKeyRequestView) Call(context view.Context) (interface{}, error) {
	// receive request
	session := session2.JSON(context)
	request := &LookupKeyRequest{}
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
	if err := session.Send(&LookupKeyResponse{Raw: ppRaw}); err != nil {
		return nil, errors.Wrapf(err, "failed to send response")
	}
	return nil, nil
}
