/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package ttxcc

import (
	"strconv"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracker/metrics"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
)

type acceptView struct {
	tx *Transaction
	id view.Identity
}

func (s *acceptView) Call(context view.Context) (interface{}, error) {
	agent := metrics.Get(context)
	agent.EmitKey(0, "ttxcc", "start", "acceptView", s.tx.ID())
	defer agent.EmitKey(0, "ttxcc", "end", "acceptView", s.tx.ID())

	// Processes
	env := s.tx.Payload.Envelope
	if env == nil {
		return nil, errors.Errorf("expected fabric envelope")
	}

	agent.EmitKey(0, "ttxcc", "start", "acceptViewStoreTransient", s.tx.ID())
	err := s.tx.storeTransient()
	if err != nil {
		return nil, errors.Wrapf(err, "failed storing transient")
	}
	agent.EmitKey(0, "ttxcc", "end", "acceptViewStoreTransient", s.tx.ID())

	agent.EmitKey(0, "ttxcc", "start", "acceptViewParseRWS", s.tx.ID())
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("parse rws for id [%s]", s.tx.ID())
	}
	ch := network.GetInstance(context, s.tx.Network(), s.tx.Channel())
	rws, err := ch.GetRWSet(s.tx.ID(), env.Results())
	if err != nil {
		return nil, errors.WithMessagef(err, "failed getting rwset for tx [%s]", s.tx.ID())
	}
	rws.Done()
	agent.EmitKey(0, "ttxcc", "end", "acceptViewParseRWS", s.tx.ID())

	agent.EmitKey(0, "ttxcc", "start", "acceptViewStoreEnv", s.tx.ID())
	rawEnv, err := env.Bytes()
	if err != nil {
		return nil, errors.WithMessagef(err, "failed marshalling tx env [%s]", s.tx.ID())
	}

	if err := ch.StoreEnvelope(env.TxID(), rawEnv); err != nil {
		return nil, errors.WithMessagef(err, "failed storing tx env [%s]", s.tx.ID())
	}
	agent.EmitKey(0, "ttxcc", "end", "acceptViewStoreEnv", s.tx.ID())

	agent.EmitKey(0, "ttxcc", "size", "acceptViewEnvelopeSize", s.tx.ID(), strconv.Itoa(len(rawEnv)))

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("send back ack")
	}
	// Ack for distribution
	session := context.Session()
	// Send the proposal response back
	err = session.Send([]byte("ack"))
	if err != nil {
		return nil, err
	}
	agent.EmitKey(0, "ttxcc", "sent", "txAck", s.tx.ID())

	return s.tx, nil
}

func NewAcceptView(tx *Transaction) *acceptView {
	return &acceptView{tx: tx}
}
