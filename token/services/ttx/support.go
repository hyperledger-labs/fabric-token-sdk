/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"encoding/base64"
	"strconv"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/comm"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracker/metrics"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

// StoreEnvelope stores the transaction envelope locally
func StoreEnvelope(context view.Context, tx *Transaction) error {
	agent := metrics.Get(context)

	agent.EmitKey(0, "ttx", "start", "acceptViewParseRWS", tx.ID())
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("parse rws for id [%s]", tx.ID())
	}
	env := tx.Payload.Envelope
	backend := network.GetInstance(context, tx.Network(), tx.Channel())
	rws, err := backend.GetRWSet(tx.ID(), env.Results())
	if err != nil {
		return errors.WithMessagef(err, "failed getting rwset for tx [%s]", tx.ID())
	}
	rws.Done()
	agent.EmitKey(0, "ttx", "end", "acceptViewParseRWS", tx.ID())

	agent.EmitKey(0, "ttx", "start", "acceptViewStoreEnv", tx.ID())
	rawEnv, err := env.Bytes()
	if err != nil {
		return errors.WithMessagef(err, "failed marshalling tx env [%s]", tx.ID())
	}

	if err := backend.StoreEnvelope(env.TxID(), rawEnv); err != nil {
		return errors.WithMessagef(err, "failed storing tx env [%s]", tx.ID())
	}
	agent.EmitKey(0, "ttx", "end", "acceptViewStoreEnv", tx.ID())

	agent.EmitKey(0, "ttx", "size", "acceptViewEnvelopeSize", tx.ID(), strconv.Itoa(len(rawEnv)))

	return nil
}

// StoreTransactionRecords stores the transaction records extracted from the passed transaction to the
// token transaction db
func StoreTransactionRecords(context view.Context, tx *Transaction) error {
	return NewOwner(context, tx.TokenRequest.TokenService).Append(tx)
}

// RunView runs passed view within the passed context and using the passed options in a separate goroutine
func RunView(context view.Context, view view.View, opts ...view.RunViewOption) {
	defer func() {
		if r := recover(); r != nil {
			logger.Debugf("panic in RunView: %v", r)
		}
	}()
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Debugf("panic in RunView: %v", r)
			}
		}()
		_, err := context.RunView(view, opts...)
		if err != nil {
			logger.Errorf("failed to run view: %s", err)
		}
	}()
}

type LocalBidirectionalChannel struct {
	left  view.Session
	right view.Session
}

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

func (c *LocalBidirectionalChannel) LeftSession() view.Session {
	return c.left
}

func (c *LocalBidirectionalChannel) RightSession() view.Session {
	return c.right
}

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
	logger.Debugf("[%s] Sending message to self session of length %d", s.name, len(payload))
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
	logger.Debugf("[%s] Sending error message to self session of length %d", s.name, len(payload))
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
	return s.readChannel
}

func (s *localSession) Close() {
	s.info.Closed = true
}
