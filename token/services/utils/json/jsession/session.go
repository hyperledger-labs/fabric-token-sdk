/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package jsession

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
)

// Session local alias for view.Sessions
type Session = view.Session

type JSession struct {
	s       Session
	context context.Context
}

// New returns a JSON-based session wrapping the passed session.
func New(context context.Context, s Session) *JSession {
	span := trace.SpanFromContext(context)
	span.AddEvent(fmt.Sprintf("Open json session to [%s:%s]", string(s.Info().Caller), s.Info().CallerViewID))
	return &JSession{s: s, context: context}
}

// NewJSON opens a new JSON-based session to the passed party using the passed caller.
func NewJSON(context view.Context, caller view.View, party view.Identity) (*JSession, error) {
	s, err := context.GetSession(caller, party)
	if err != nil {
		return nil, err
	}
	return NewFromSession(context, s), nil
}

// NewFromInitiator opens a new JSON-based session to the passed party
func NewFromInitiator(context view.Context, party view.Identity) (*JSession, error) {
	s, err := context.GetSession(context.Initiator(), party)
	if err != nil {
		return nil, err
	}
	return NewFromSession(context, s), nil
}

// NewFromSession returns a new JSON-based session wrapping the passed session
func NewFromSession(context view.Context, s Session) *JSession {
	return New(context.Context(), s)
}

// FromContext return a new JSON-based sessions wrapping the context's session
func FromContext(context view.Context) *JSession {
	return New(context.Context(), context.Session())
}

func (j *JSession) Receive(state interface{}) error {
	return j.ReceiveWithTimeout(state, 0)
}

func (j *JSession) ReceiveWithTimeout(state interface{}, d time.Duration) error {
	raw, err := j.ReceiveRawWithTimeout(d)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, state)
}

func (j *JSession) ReceiveRaw() ([]byte, error) {
	return j.ReceiveRawWithTimeout(0)
}

func (j *JSession) ReceiveRawWithTimeout(d time.Duration) ([]byte, error) {
	span := trace.SpanFromContext(j.context)

	current := j.context
	if d != 0 {
		var cancel context.CancelFunc
		current, cancel = context.WithTimeout(j.context, d)
		defer cancel()
	}

	span.AddEvent("Wait to receive")
	ch := j.s.Receive()
	var raw []byte
	select {
	case msg := <-ch:
		span.AddEvent("Received message")
		if msg == nil {
			return nil, errors.New("received nil message")
		}
		if msg.Status == view.ERROR {
			return nil, errors.Errorf("received error from remote [%s]", string(msg.Payload))
		}
		raw = msg.Payload
	case <-current.Done():
		span.AddEvent("Context done")
		err := errors.Errorf("context done [%s]", j.context.Err())
		span.RecordError(err)
		return nil, err
	}
	return raw, nil
}

func (j *JSession) Send(state interface{}) error {
	return j.SendWithContext(j.context, state)
}

func (j *JSession) SendWithContext(ctx context.Context, state interface{}) error {
	v, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return j.s.SendWithContext(ctx, v)
}

func (j *JSession) SendRaw(ctx context.Context, raw []byte) error {
	return j.s.SendWithContext(ctx, raw)
}

func (j *JSession) SendError(err string) error {
	return j.SendErrorWithContext(j.context, err)
}

func (j *JSession) SendErrorWithContext(ctx context.Context, err string) error {
	span := trace.SpanFromContext(ctx)
	span.RecordError(errors.New(err))
	return j.s.SendErrorWithContext(ctx, []byte(err))
}

func (j *JSession) Session() Session {
	return j.s
}

func (j *JSession) Info() view.SessionInfo {
	return j.s.Info()
}
