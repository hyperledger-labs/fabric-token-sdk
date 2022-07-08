/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"encoding/base64"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/comm"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
)

type selfSession struct {
	id        string
	caller    string
	contextID string
	endpoint  string
	pkid      []byte
	info      view.SessionInfo
	ch        chan *view.Message
}

func newSelfSession(caller string, contextID string, endpoint string, pkid []byte) (*selfSession, error) {
	ID, err := comm.GetRandomNonce()
	if err != nil {
		return nil, err
	}

	return &selfSession{
		id:        base64.StdEncoding.EncodeToString(ID),
		caller:    caller,
		contextID: contextID,
		endpoint:  endpoint,
		pkid:      pkid,
		info: view.SessionInfo{
			ID:           base64.StdEncoding.EncodeToString(ID),
			Caller:       nil,
			CallerViewID: "",
			Endpoint:     endpoint,
			EndpointPKID: pkid,
			Closed:       false,
		},
		ch: make(chan *view.Message, 1),
	}, nil
}

func (s *selfSession) Info() view.SessionInfo {
	return s.info
}

func (s *selfSession) Send(payload []byte) error {
	logger.Debugf("Sending message to self session of length %d", len(payload))
	s.ch <- &view.Message{
		SessionID:    s.info.ID,
		ContextID:    s.contextID,
		Caller:       s.caller,
		FromEndpoint: s.endpoint,
		FromPKID:     s.pkid,
		Status:       view.OK,
		Payload:      payload,
	}
	return nil
}

func (s *selfSession) SendError(payload []byte) error {
	logger.Debugf("Sending error message to self session of length %d", len(payload))
	s.ch <- &view.Message{
		SessionID:    s.info.ID,
		ContextID:    s.contextID,
		Caller:       s.caller,
		FromEndpoint: s.endpoint,
		FromPKID:     s.pkid,
		Status:       view.ERROR,
		Payload:      payload,
	}
	return nil
}

func (s *selfSession) Receive() <-chan *view.Message {
	return s.ch
}

func (s *selfSession) Close() {
	s.info.Closed = true
}
