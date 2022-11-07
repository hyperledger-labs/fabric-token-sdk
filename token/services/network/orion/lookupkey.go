/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package orion

import (
	"encoding/base64"
	"fmt"
	"time"

	orion2 "github.com/hyperledger-labs/fabric-smart-client/platform/orion"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	session2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/session"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

type LookupKeyRequest struct {
	Network      string
	Namespace    string
	StartingTxID string
	Key          string
	Timeout      time.Duration
}

func (l *LookupKeyRequest) String() string {
	return fmt.Sprintf("[%s:%s:%s:%s]", l.Network, l.Namespace, l.StartingTxID, l.Key)
}

type LookupKeyResponse struct {
	Raw []byte
}

type LookupKeyRequestView struct {
	*LookupKeyRequest
}

func NewLookupKeyRequestView(network string, namespace string, startingTxID string, key string, timeout time.Duration) *LookupKeyRequestView {
	return &LookupKeyRequestView{
		LookupKeyRequest: &LookupKeyRequest{
			Network:      network,
			Namespace:    namespace,
			StartingTxID: startingTxID,
			Key:          key,
			Timeout:      timeout,
		},
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
		return LookupKey(context, v.LookupKeyRequest)
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
		return nil, errors.Wrapf(err, "failed to get session to custodian [%s] for request [%s]", custodian, v.LookupKeyRequest)
	}
	if err := session.Send(v.LookupKeyRequest); err != nil {
		return nil, errors.Wrapf(err, "failed to send request [%s] to custodian [%s]", v.LookupKeyRequest, custodian)
	}
	response := &LookupKeyResponse{}
	if err := session.ReceiveWithTimeout(response, v.Timeout); err != nil {
		return nil, errors.Wrapf(err, "failed to receive response from custodian [%s] on request [%s]", custodian, v.LookupKeyRequest)
	}
	return response.Raw, nil
}

type LookupKeyRequestRespondView struct{}

func (v *LookupKeyRequestRespondView) Call(context view.Context) (interface{}, error) {
	// receive request
	session := session2.JSON(context)
	request := &LookupKeyRequest{}
	if err := session.Receive(request); err != nil {
		return nil, errors.Wrapf(err, "failed to receive request")
	}
	logger.Debugf("request: %+v", request)

	// Get key's value
	value, err := LookupKey(context, request)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get key's value from orion network [%s]", request.Network)
	}
	logger.Debugf("get key's value, done: %d", len(value))
	if err := session.Send(&LookupKeyResponse{Raw: value}); err != nil {
		return nil, errors.Wrapf(err, "failed to send response")
	}
	return nil, nil
}

func LookupKey(context view.Context, request *LookupKeyRequest) ([]byte, error) {
	ons := orion2.GetOrionNetworkService(context, request.Network)
	if ons == nil {
		return nil, errors.Errorf("cannot find orion netwotk [%s]", request.Network)
	}
	custodianID, err := GetCustodian(view2.GetConfigService(context), request.Network)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get custodian identifier")
	}
	s, err := ons.SessionManager().NewSession(custodianID)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get a new session")
	}
	qe, err := s.QueryExecutor(request.Namespace)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get query executor for [%s:%s]", request.Network, request.Namespace)
	}

	pollingTime := int64(500)
	iterations := int(request.Timeout.Milliseconds() / pollingTime)
	if iterations == 0 {
		iterations = 1
	}
	for i := 0; i < iterations; i++ {
		timeout := time.NewTimer(time.Duration(pollingTime) * time.Millisecond)

		stop := false
		select {
		case <-context.Context().Done():
			timeout.Stop()
			return nil, errors.Errorf("view context done")
		case <-timeout.C:
			timeout.Stop()
			v, err := qe.Get(request.Key)
			if err != nil {
				logger.Errorf("failed to get key [%s] from [%s:%s]", request.Key, request.Network, request.Namespace)
			}
			logger.Debugf("get key [%s] from [%s:%s], result [%d]", request.Key, request.Network, request.Namespace, len(v))
			if len(v) != 0 {
				if logger.IsEnabledFor(zapcore.DebugLevel) {
					logger.Debugf("scanning for key [%s] with timeout [%s] found, [%s]",
						request.Key,
						timeout,
						base64.StdEncoding.EncodeToString(v),
					)
				}
				return v, nil
			}
		}
		if stop {
			break
		}
	}
	return nil, errors.Errorf("cannot find get key [%s] from [%s:%s]", request.Key, request.Network, request.Namespace)
}
