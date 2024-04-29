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

type BroadcastRequest struct {
	Network string
	Blob    []byte
}

type BroadcastResponse struct {
	Err error
}

type BroadcastView struct {
	Network driver.Network
	Blob    interface{}
}

func NewBroadcastView(network driver.Network, blob interface{}) *BroadcastView {
	return &BroadcastView{Network: network, Blob: blob}
}

func (r *BroadcastView) Call(context view.Context) (interface{}, error) {
	custodian, err := GetCustodian(view2.GetConfigService(context), r.Network.Name())
	if err != nil {
		return nil, errors.Wrap(err, "failed to get custodian identifier")
	}
	logger.Debugf("custodian: %s", custodian)
	session, err := session2.NewJSON(context, context.Initiator(), view2.GetIdentityProvider(context).Identity(custodian))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get session to custodian [%s]", custodian)
	}

	var blob []byte
	switch b := r.Blob.(type) {
	case driver.Envelope:
		var err error
		blob, err = b.Bytes()
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal envelope")
		}
	default:
		return nil, errors.Errorf("unsupported blob type [%T]", b)
	}

	// TODO: Should we sign the broadcast request?
	request := &BroadcastRequest{
		Network: r.Network.Name(),
		Blob:    blob,
	}
	if err := session.Send(request); err != nil {
		return nil, errors.Wrapf(err, "failed to send request to custodian [%s]", custodian)
	}
	response := &BroadcastResponse{}
	if err := session.Receive(response); err != nil {
		return nil, errors.Wrapf(err, "failed to receive response from custodian [%s]", custodian)
	}
	if response.Err != nil {
		return nil, errors.Wrapf(response.Err, "failed to broadcast")
	}
	return nil, nil
}

type BroadcastResponderView struct{}

func (r *BroadcastResponderView) Call(context view.Context) (interface{}, error) {
	// receive request
	session := session2.JSON(context)
	request := &BroadcastRequest{}
	if err := session.Receive(request); err != nil {
		return nil, errors.Wrapf(err, "failed to receive request")
	}
	logger.Debugf("request: %+v", request)

	// commit
	ons, err := orion.GetOrionNetworkService(context, request.Network)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get orion network service for network [%s]", request.Network)
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
	tm := ons.TransactionManager()
	env := tm.NewEnvelope()
	if err := env.FromBytes(request.Blob); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal envelope")
	}
	logger.Debugf("commit envelope [%s][%s]", env.TxID(), env.String())
	if err := ons.TransactionManager().CommitEnvelope(oSession, env); err != nil {
		return nil, errors.Wrap(err, "failed to commit envelope")
	}

	// all good
	if err := session.Send(&BroadcastResponse{}); err != nil {
		return nil, errors.Wrapf(err, "failed to send response")
	}

	return nil, nil
}
