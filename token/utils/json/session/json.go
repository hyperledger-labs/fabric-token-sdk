/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package session

import (
	"context"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/comm/session"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
)

type Session = view.Session

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

func NewJSON(context view.Context, caller view.View, party view.Identity) (JsonSession, error) {
	return session.NewJSON(context, caller, party)
}

func NewFromInitiator(context view.Context, party view.Identity) (JsonSession, error) {
	return session.NewFromInitiator(context, party)
}

func NewFromSession(context view.Context, s Session) JsonSession {
	return session.NewFromSession(context, s)
}

func JSON(context view.Context) JsonSession {
	return session.JSON(context)
}
