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

type jSession struct {
	s       Session
	context context.Context
}

func newJSONSession(s Session, context context.Context) *jSession {
	span := trace.SpanFromContext(context)
	span.AddEvent(fmt.Sprintf("Open json session to [%s:%s]", string(s.Info().Caller), s.Info().CallerViewID))
	return &jSession{s: s, context: context}
}

func (j *jSession) Receive(state interface{}) error {
	return j.ReceiveWithTimeout(state, 0)
}

func (j *jSession) ReceiveWithTimeout(state interface{}, d time.Duration) error {
	raw, err := j.ReceiveRawWithTimeout(d)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, state)
}

func (j *jSession) ReceiveRaw() ([]byte, error) {
	return j.ReceiveRawWithTimeout(0)
}

func (j *jSession) ReceiveRawWithTimeout(d time.Duration) ([]byte, error) {
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

func (j *jSession) Send(state interface{}) error {
	return j.SendWithContext(j.context, state)
}

func (j *jSession) SendWithContext(ctx context.Context, state interface{}) error {
	v, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return j.s.SendWithContext(ctx, v)
}

func (j *jSession) SendRaw(ctx context.Context, raw []byte) error {
	return j.s.SendWithContext(ctx, raw)
}

func (j *jSession) SendError(err string) error {
	return j.SendErrorWithContext(j.context, err)
}

func (j *jSession) SendErrorWithContext(ctx context.Context, err string) error {
	span := trace.SpanFromContext(ctx)
	span.RecordError(errors.New(err))
	return j.s.SendErrorWithContext(ctx, []byte(err))
}

func (j *jSession) Session() Session {
	return j.s
}

func (j *jSession) Info() view.SessionInfo {
	return j.s.Info()
}
