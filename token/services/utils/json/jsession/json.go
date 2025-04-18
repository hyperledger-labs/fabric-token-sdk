/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package jsession

import (
	"context"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
)

type Session = view.Session

type JSession interface {
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

func NewJSON(context view.Context, caller view.View, party view.Identity) (JSession, error) {
	s, err := context.GetSession(caller, party)
	if err != nil {
		return nil, err
	}
	return NewFromSession(context, s), nil
}

func NewFromInitiator(context view.Context, party view.Identity) (JSession, error) {
	s, err := context.GetSession(context.Initiator(), party)
	if err != nil {
		return nil, err
	}
	return NewFromSession(context, s), nil
}

func NewFromSession(context view.Context, s Session) JSession {
	return newJSONSession(s, context.Context())
}

func FromContext(context view.Context) JSession {
	return newJSONSession(context.Session(), context.Context())
}
