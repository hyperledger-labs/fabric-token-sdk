/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"encoding/base64"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/comm"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/pkg/errors"
)

// LocalBidirectionalChannel is a bidirectional channel that is used to simulate
// a session between two views (let's call them L and R) running in the same process.
type LocalBidirectionalChannel struct {
	left  view.Session
	right view.Session
}

// NewLocalBidirectionalChannel creates a new bidirectional channel
func NewLocalBidirectionalChannel(caller string, contextID string, endpoint string, pkid []byte) (*LocalBidirectionalChannel, error) {
	ID, err := comm.GetRandomNonce()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate session ID")
	}
	lr := make(chan *view.Message, 10)
	rl := make(chan *view.Message, 10)

	info := view.SessionInfo{
		ID:           base64.StdEncoding.EncodeToString(ID),
		Caller:       nil,
		CallerViewID: "",
		Endpoint:     endpoint,
		EndpointPKID: pkid,
		Closed:       false,
	}
	return &LocalBidirectionalChannel{
		left: &localSession{
			name:         "left",
			contextID:    contextID,
			caller:       caller,
			info:         info,
			readChannel:  rl,
			writeChannel: lr,
		},
		right: &localSession{
			name:         "right",
			contextID:    contextID,
			caller:       caller,
			info:         info,
			readChannel:  lr,
			writeChannel: rl,
		},
	}, nil
}

// LeftSession returns the session from the L to R
func (c *LocalBidirectionalChannel) LeftSession() view.Session {
	return c.left
}

// RightSession returns the session from the R to L
func (c *LocalBidirectionalChannel) RightSession() view.Session {
	return c.right
}

// localSession is a local session that is used to simulate a session between two views.
// It has a read channel and a write channel.
type localSession struct {
	name         string
	contextID    string
	caller       string
	info         view.SessionInfo
	readChannel  chan *view.Message
	writeChannel chan *view.Message
}

func (s *localSession) Info() view.SessionInfo {
	return s.info
}

func (s *localSession) Send(payload []byte) error {
	if s.info.Closed {
		return errors.New("session is closed")
	}

	s.writeChannel <- &view.Message{
		SessionID:    s.info.ID,
		ContextID:    s.contextID,
		Caller:       s.caller,
		FromEndpoint: s.info.Endpoint,
		FromPKID:     s.info.EndpointPKID,
		Status:       view.OK,
		Payload:      payload,
	}
	return nil
}

func (s *localSession) SendError(payload []byte) error {
	if s.info.Closed {
		return errors.New("session is closed")
	}

	s.writeChannel <- &view.Message{
		SessionID:    s.info.ID,
		ContextID:    s.contextID,
		Caller:       s.caller,
		FromEndpoint: s.info.Endpoint,
		FromPKID:     s.info.EndpointPKID,
		Status:       view.ERROR,
		Payload:      payload,
	}
	return nil
}

func (s *localSession) Receive() <-chan *view.Message {
	if s.info.Closed {
		return nil
	}
	return s.readChannel
}

func (s *localSession) Close() {
	s.info.Closed = true
}
