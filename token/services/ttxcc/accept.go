/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package ttxcc

import (
	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
)

type acceptView struct {
	tx *Transaction
	id view.Identity
}

func (s *acceptView) Call(context view.Context) (interface{}, error) {
	// Processes
	env := s.tx.Payload.FabricEnvelope
	if env == nil {
		return nil, errors.Errorf("expected fabric envelope")
	}
	err := s.tx.storeTransient()
	if err != nil {
		return nil, errors.Wrapf(err, "failed storing transient")
	}

	logger.Debugf("parse rws for id [%s]", s.tx.ID())
	ch := fabric.GetChannel(context, s.tx.Network(), s.tx.Channel())
	rws, err := ch.Vault().GetRWSet(s.tx.ID(), env.Results())
	if err != nil {
		return nil, errors.WithMessagef(err, "failed getting rwset for tx [%s]", s.tx.ID())
	}
	rws.Done()

	rawEnv, err := env.Bytes()
	if err != nil {
		return nil, errors.WithMessagef(err, "failed marshalling tx env [%s]", s.tx.ID())
	}

	if err := ch.Vault().StoreEnvelope(env.TxID(), rawEnv); err != nil {
		return nil, errors.WithMessagef(err, "failed storing tx env [%s]", s.tx.ID())
	}

	logger.Debugf("send back ack")
	// Ack for distribution
	session := context.Session()
	// Send the proposal response back
	err = session.Send([]byte("ack"))
	if err != nil {
		return nil, err
	}

	return s.tx, nil
}

func NewAcceptView(tx *Transaction) *acceptView {
	return &acceptView{tx: tx}
}
