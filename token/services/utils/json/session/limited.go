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
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
)

// SizeLimitedJsonSession wraps a JsonSession and rejects any incoming message
// whose raw byte length exceeds maxBytes. The size check fires before JSON
// deserialization so that oversized payloads are dropped without ever
// allocating the decoded struct, preventing memory exhaustion attacks.
//
// This is the recommended way to enforce per-service message-size limits
// without modifying each view's Call implementation individually.
type SizeLimitedJsonSession struct {
	inner    JsonSession
	maxBytes int
}

// NewSizeLimitedSession wraps inner with a size guard. Any received message
// whose raw wire length exceeds maxBytes is rejected before deserialization.
func NewSizeLimitedSession(inner JsonSession, maxBytes int) JsonSession {
	return &SizeLimitedJsonSession{inner: inner, maxBytes: maxBytes}
}

// JSONWithLimit returns a JsonSession backed by the current FSC session that
// enforces an upper bound on received message sizes. Any message whose raw
// wire length exceeds maxBytes is rejected before JSON deserialization.
func JSONWithLimit(ctx view.Context, maxBytes int) JsonSession {
	return NewSizeLimitedSession(JSON(ctx), maxBytes)
}

// ReceiveRaw reads the next raw message and returns an error if its length
// exceeds the configured limit.
func (s *SizeLimitedJsonSession) ReceiveRaw() ([]byte, error) {
	raw, err := s.inner.ReceiveRaw()
	if err != nil {
		return nil, err
	}
	if err := s.checkSize(raw); err != nil {
		return nil, err
	}

	return raw, nil
}

// ReceiveRawWithTimeout reads the next raw message with a deadline and returns
// an error if its length exceeds the configured limit.
func (s *SizeLimitedJsonSession) ReceiveRawWithTimeout(d time.Duration) ([]byte, error) {
	raw, err := s.inner.ReceiveRawWithTimeout(d)
	if err != nil {
		return nil, err
	}
	if err := s.checkSize(raw); err != nil {
		return nil, err
	}

	return raw, nil
}

// Receive reads the next message, enforces the size limit, then unmarshals it
// into state.
func (s *SizeLimitedJsonSession) Receive(state interface{}) error {
	raw, err := s.ReceiveRaw()
	if err != nil {
		return err
	}

	return json.Unmarshal(raw, state)
}

// ReceiveWithTimeout reads the next message with a deadline, enforces the size
// limit, then unmarshals it into state.
func (s *SizeLimitedJsonSession) ReceiveWithTimeout(state interface{}, d time.Duration) error {
	raw, err := s.ReceiveRawWithTimeout(d)
	if err != nil {
		return err
	}

	return json.Unmarshal(raw, state)
}

// The remaining methods delegate unchanged to the inner session.

func (s *SizeLimitedJsonSession) Info() view.SessionInfo { return s.inner.Info() }

func (s *SizeLimitedJsonSession) Send(payload any) error { return s.inner.Send(payload) }

func (s *SizeLimitedJsonSession) SendRaw(ctx context.Context, raw []byte) error {
	return s.inner.SendRaw(ctx, raw)
}

func (s *SizeLimitedJsonSession) SendWithContext(ctx context.Context, payload any) error {
	return s.inner.SendWithContext(ctx, payload)
}

func (s *SizeLimitedJsonSession) SendError(msg string) error { return s.inner.SendError(msg) }

func (s *SizeLimitedJsonSession) SendErrorWithContext(ctx context.Context, msg string) error {
	return s.inner.SendErrorWithContext(ctx, msg)
}

func (s *SizeLimitedJsonSession) Session() Session { return s.inner.Session() }

func (s *SizeLimitedJsonSession) checkSize(raw []byte) error {
	if len(raw) > s.maxBytes {
		return errors.Errorf("message too large (%d > %d bytes)", len(raw), s.maxBytes)
	}

	return nil
}
