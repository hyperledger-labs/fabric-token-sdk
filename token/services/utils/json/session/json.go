/*
Copyright IBM Corp. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package session

import (
	"context"
	"encoding/json"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/services/logging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
)

const defaultReceiveTimeout = 10 * time.Second

// ErrTimeout is returned when a session receive operation times out.
var ErrTimeout = errors.New("session timeout")

var logger = logging.MustGetLogger()

type Session = view.Session

// Marshaller is a generic interface for marshalling and unmarshalling data.
type Marshaller interface {
	Marshal(v any) ([]byte, error)
	Unmarshal(data []byte, v any) error
}

// JSONMarshaller is the default JSON marshaller.
type JSONMarshaller struct{}

func (JSONMarshaller) Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (JSONMarshaller) Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

type jsonSession struct {
	s          Session
	ctx        context.Context
	marshaller Marshaller
}

type JsonSession interface {
	Info() view.SessionInfo
	Send(payload any) error
	SendRaw(ctx context.Context, raw []byte) error
	SendWithContext(ctx context.Context, payload any) error
	SendError(error string) error
	SendErrorWithContext(ctx context.Context, error string) error
	Receive(state interface{}) error
	ReceiveWithTimeout(state interface{}, d time.Duration) error
	ReceiveRaw() ([]byte, error)
	ReceiveRawWithTimeout(d time.Duration) ([]byte, error)
	Session() Session
}

func NewJSON(viewCtx view.Context, caller view.View, party view.Identity) (JsonSession, error) {
	s, err := viewCtx.GetSession(caller, party)
	if err != nil {
		return nil, err
	}

	return NewFromSession(viewCtx, s), nil
}

func NewFromInitiator(viewCtx view.Context, party view.Identity) (JsonSession, error) {
	s, err := viewCtx.GetSession(viewCtx.Initiator(), party)
	if err != nil {
		return nil, err
	}

	return NewFromSession(viewCtx, s), nil
}

func NewFromSession(viewCtx view.Context, session Session) JsonSession {
	return newJSONSession(session, viewCtx.Context(), JSONMarshaller{})
}

func JSON(viewCtx view.Context) JsonSession {
	return newJSONSession(viewCtx.Session(), viewCtx.Context(), JSONMarshaller{})
}

func newJSONSession(s Session, ctx context.Context, m Marshaller) *jsonSession {
	logger.DebugfContext(ctx, "Open json session to [%s]", logging.Eval(s.Info))

	return &jsonSession{s: s, ctx: ctx, marshaller: m}
}

func (j *jsonSession) Receive(state interface{}) error {
	return j.ReceiveWithTimeout(state, defaultReceiveTimeout)
}

func (j *jsonSession) ReceiveWithTimeout(state interface{}, d time.Duration) error {
	raw, err := j.ReceiveRawWithTimeout(d)
	if err != nil {
		return err
	}
	err = j.marshaller.Unmarshal(raw, state)
	if err != nil {
		return errors.Wrapf(err, "failed unmarshalling state, len [%d]", len(raw))
	}
	return nil
}

func (j *jsonSession) ReceiveRaw() ([]byte, error) {
	return j.ReceiveRawWithTimeout(defaultReceiveTimeout)
}

func (j *jsonSession) ReceiveRawWithTimeout(d time.Duration) ([]byte, error) {
	timeout := time.NewTimer(d)
	defer timeout.Stop()

	logger.DebugfContext(j.ctx, "Wait to receive")
	ch := j.s.Receive()
	select {
	case msg := <-ch:
		if msg == nil {
			logger.ErrorfContext(j.ctx, "Received nil message")
			return nil, errors.New("received message is nil")
		}
		if msg.Status == view.ERROR {
			logger.ErrorfContext(j.ctx, "Received error message")
			return nil, errors.Errorf("received error from remote [%s]", string(msg.Payload))
		}
		logger.DebugfContext(j.ctx, "json session, received message [%s]", logging.SHA256Base64(msg.Payload))
		return msg.Payload, nil
	case <-timeout.C:
		logger.ErrorfContext(j.ctx, "timeout reached")
		return nil, errors.Join(errors.Errorf("time out reached on session [%s]", j.Info().ID), ErrTimeout)
	case <-j.ctx.Done():
		logger.ErrorfContext(j.ctx, "ctx done: %w", j.ctx.Err())
		return nil, errors.Errorf("ctx done [%s]", j.ctx.Err())
	}
}

func (j *jsonSession) Send(state interface{}) error {
	return j.SendWithContext(j.ctx, state)
}

func (j *jsonSession) SendWithContext(ctx context.Context, state interface{}) error {
	v, err := j.marshaller.Marshal(state)
	if err != nil {
		return err
	}
	logger.DebugfContext(ctx, "json session, send message [%s]", logging.SHA256Base64(v))
	return j.s.SendWithContext(ctx, v)
}

func (j *jsonSession) SendRaw(ctx context.Context, raw []byte) error {
	logger.DebugfContext(ctx, "json session, send raw message [%s]", logging.SHA256Base64(raw))
	return j.s.SendWithContext(ctx, raw)
}

func (j *jsonSession) SendError(err string) error {
	return j.SendErrorWithContext(j.ctx, err)
}

func (j *jsonSession) SendErrorWithContext(ctx context.Context, err string) error {
	logger.ErrorfContext(ctx, "json session, send error: %w", err)
	return j.s.SendErrorWithContext(ctx, []byte(err))
}

func (j *jsonSession) Session() Session {
	return j.s
}

func (j *jsonSession) Info() view.SessionInfo {
	return j.s.Info()
}
